SRC=$1
CN=pkg/lcp/canonical.go; LRE=x
run(){ name=$1; expr=$5; mkdir -p "m_$name"; sed "s/^package lcp/package m_$name/" $SRC > "m_$name/c.go"; sed -i "$expr" "m_$name/c.go"; if cmp -s "m_$name/c.go" <(sed "s/^package lcp/package m_$name/" $SRC); then echo "NOT APPLIED $name"; fi; }
. ./rows.sh
