# Helix Reader — Implemented Architecture

## Scope

These diagrams describe current code paths and explicit integration boundaries. Dashed nodes are prototypes, optional components, or unfinished integrations; they are not production capabilities.

## Runtime topology

```mermaid
flowchart LR
    P[Responsive React prototype\nsample memory only] -. no API connection .-> API
    R[Third-party OPDS readers] -->|OPDS 1.2 / 2.0| API
    K[KOReader clients] -->|account + document progress| API
    A[Admin/API client] -->|Bearer token| API
    API[Go/Gin HTTP service\nHTTP/1.1; HTTP/2 or HTTP/3 with TLS] --> C{Catalog repository}
    C -->|configured| PG[(PostgreSQL)]
    C -->|fallback with warning| MEM[(Volatile memory)]
    API -->|progress + KOReader sync| RD[(Redis)]
    API -->|optional vector memory| PV[(pgvector)]
    API --> CC[Validated cover cache]
    API --> ING[Read-only ingestion]
    ING --> SRC[Local / mounts / HTTP(S)\nWebDAV / OPDS / rclone]
```

The web prototype does not call the API. Native desktop/mobile applications, real content renderers, external conversion engines, OIDC, Kobo sync, OPDS-PSE, and LCP fulfillment are not shown as connected because they are not implemented.

## Catalog identity

```mermaid
erDiagram
    WORK ||--|{ EDITION : groups
    EDITION ||--|{ FILE : has
    EDITION ||--|{ IDENTIFIER : identifies
    EDITION ||--|{ METADATA_EVIDENCE : supports
    WORK {
      string id
      string canonical_title
      string series
      float series_index
    }
    EDITION {
      string id
      string title
      string identification_state
      float identification_confidence
    }
    FILE {
      string id
      string source_id
      string format
      string sha256
      bool protected
    }
    IDENTIFIER {
      string scheme
      string value
      string source
      float confidence
    }
    METADATA_EVIDENCE {
      string field
      string provider
      float confidence
      time observed_at
    }
```

ISBNs and other edition identifiers are not collapsed onto the work as metadata fields. Work and edition IDs use a deterministically selected canonical ISBN-13 when available, with content SHA-256 as the no-valid-ISBN fallback. ISBN-10/13 equivalents, hyphenation, scheme labels, and identifier ordering cannot fork identity, whether or not the optional identification pipeline runs; every file independently retains its byte hash and stable, case-sensitive source/path ID. Different byte streams carrying the same ISBN therefore share one work/edition as multiple files instead of colliding. When no-ISBN bytes change, the source/path file is re-homed to the new content work and an emptied old work is removed. Ingestion merges through one atomic repository operation. PostgreSQL serializes cross-work identity writes with a transaction-scoped advisory lock; the memory repository applies equivalent re-homing under its write lock and returns deep copies. Successful, unprefixed scans of connectors with recursive listings atomically remove catalog files absent from the complete source snapshot, then remove editions and works emptied by that deletion. Per-source lock stripes prevent an older concurrent listing from pruning newer scan results. Prefix scans, non-recursive sources, canceled/failed listings, and listed files whose ingest fails never authorize pruning.

## Ingestion sequence

```mermaid
sequenceDiagram
    participant Admin
    participant API as Scan job API
    participant Conn as Read-only connector
    participant Parse as Bounded parser/identifier
    participant Repo as Catalog repository
    Admin->>API: POST /v1/scans {source_id, prefix}
    API-->>Admin: 202 Accepted + job ID
    API->>Conn: List(prefix)
    Conn-->>API: File entries
    loop Each recognized file
        API->>Conn: Open(path)
        Conn-->>Parse: Stream
        Parse->>Parse: SHA-256 + bounded metadata extraction
        Parse-->>API: Work/edition/file + evidence
        API->>Repo: MergeWork(work)
    end
    API-->>Admin: Job report with files, skips, warnings, errors
```

Scan jobs and source registrations currently live in process memory. They are not durable schedules or a watch-folder service. Administrators can supply a validated stable source ID and reuse it after restart to reattach existing source/path file identities; changing that ID defines a different source and can duplicate files. The runner permits two active scans and 64 unfinished jobs, retains a bounded terminal history, contains worker panics, observes queued/running cancellation, and drains workers before connector shutdown. Scan admission holds the source read lock through job registration; deletion takes the source write lock and rejects sources with unfinished scans, preventing connector closure during the admission gap or active ingestion. Indexing never requires moving or rewriting originals. Full recursive scans report the number of stale catalog files removed; OPDS pagination caps fail closed rather than returning a partial snapshot.

## OPDS surfaces

```mermaid
flowchart TB
    C[(Catalog)] --> F12[OPDS 1.2 Atom feeds]
    C --> F20[OPDS 2.0 JSON/RWPM feeds]
    F12 --> E12[Standalone edition entry]
    F20 --> E20[Standalone publication]
    F12 --> OS[OpenSearch description]
    E12 --> ACQ[Acquisition and cover links]
    E20 --> ACQ
    ACQ --> RANGE[Single-range file download]
```

OPDS-PSE links and page extraction are not currently emitted. Detection or streaming of CBZ/PDF does not establish page rendering.

## Internal progress and KOReader compatibility

```mermaid
sequenceDiagram
    participant KR as KOReader
    participant API as Helix KOSync API
    participant Auth as Account verifier
    participant Redis
    KR->>API: POST /users/create (only when enabled)
    API->>Auth: bcrypt(password digest)
    Auth->>Redis: SET NX verifier
    KR->>API: PUT /syncs/progress + x-auth headers
    API->>Redis: Read verifier
    API->>API: Constant-purpose auth path + bounded validation
    API->>API: Assign server timestamp
    API->>Redis: SET user/document progress
    KR->>API: GET /syncs/progress/{document}
    API->>Redis: Read caller's user/document key
    Redis-->>KR: Progress or 404
```

The separate internal `/v1/sync/progress` API stores structured positions and vector clocks and resolves concurrent updates deterministically. It is not wire-compatible with KOReader; the KOSync surface exists to bridge that protocol difference.

## Trust and credential boundaries

```mermaid
flowchart LR
    U[Untrusted network/client] -->|bounded headers and bodies| EDGE[Gin middleware]
    EDGE --> AUTH[Bearer / Basic / KOSync auth]
    AUTH --> HANDLER[Validated handlers]
    HANDLER -->|configured credentials| CONN[Source connector]
    CONN -->|same-origin redirects only for secrets| REMOTE[Source origin]
    CONN -. no configured secret forwarding .-> OTHER[Cross-origin acquisition]
    HANDLER -->|no secrets in errors/logs| RESP[Public response]
```

Local connectors resolve final symlink targets and enforce source-root containment. Plain HTTP/WebDAV sources reject cross-origin redirects in the redirect callback before the new origin receives a request. HTTP clients bound connection and response-header establishment without imposing a whole-body deadline on long book transfers; request contexts still cancel active streams. Remote source output and cover downloads are bounded. OPDS acquisition dials validate every resolved cross-origin address and pin the connection to the validated IP; private/loopback destinations require an explicit source opt-in, while link-local and other restricted ranges remain forbidden. Configured credentials are stripped on origin changes. OPDS catalog file paths contain only credential-free acquisition URLs; signed URLs are retained solely in the connector’s mutex-protected process-local map for the current source session and cleared on close. Distinct query-addressed books receive deterministic synthetic `helix-x-` key fragments instead of collapsing; order-independent new groups plus previous-generation key reuse keep identities stable through feed reorder and membership changes. Complete listings atomically replace/prune the raw-URL generation without exposing queries to catalog storage. Because source registrations and those mappings do not survive restart, signed remote sources must be re-registered with their stable source ID and rescanned before their content can be opened again. Scan-job errors and report locations redact URL userinfo, queries, and fragments before persistence in process-local job records. Cover downloads validate public IP targets, redirect limits, decoded image type, dimensions, and atomic cache writes. Secrets are configuration inputs, never catalog metadata or evidence content.

## AI decision boundary

```mermaid
flowchart LR
    FILE[Book bytes] --> HASH[Content identity + deterministic extraction]
    HASH --> DET[Deterministic evidence]
    FILE --> EX[Bounded excerpt]
    EX --> LLM[Configured provider integration]
    LLM --> PROP[Proposal + provenance + confidence]
    DET --> REVIEW[Review/decision model]
    PROP --> REVIEW
    REVIEW -->|accept / reject / needs review| CAT[(Catalog)]
    PROP -. cannot override stronger evidence .-> DET
```

pgvector memory is optional and dimension-checked. Current code does not establish every external LLM provider or content renderer. Provider names and strategy labels never promote a proposal to verified metadata or a format to rendered status.
