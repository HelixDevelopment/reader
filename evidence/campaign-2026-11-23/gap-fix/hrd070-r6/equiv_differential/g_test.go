package diffh

import (
	"math/rand"
	"strings"
	"testing"

	"diffh/orig"
 "diffh/m_C02_canon_decode_err_EQUIV"
 "diffh/m_C10_wtm_key_notstring_UNREACHABLE"
 "diffh/m_C11_wtm_decode_err_EQUIV"
 "diffh/m_C22_snd_key_notstring_UNREACHABLE"
 "diffh/m_C28_snd_default_UNREACHABLE"
 "diffh/m_C47_wc_default_UNREACHABLE"
 "diffh/m_C63_cn_first_empty_digits_EQUIV"
 "diffh/m_C64_cn_second_empty_digits_UNREACHABLE"
)

// Differential harness for the HRD-070 equivalent-mutant arguments: the original canonical.go and each
// candidate-equivalent mutant (same sed expressions as mutate.sh) are run on the same documents through
// Canonical, CanonicalAsReference and CanonicalForSigning; any difference in (output, error text) is reported.
func s(b []byte, e error) string {
	if e != nil {
		return "ERR " + e.Error()
	}
	return "OK " + string(b)
}

type fn = func([]byte) ([]byte, error)

var muts = []struct {
	n       string
	a, b, c fn
}{
	{"C02_canon_decode_err_EQUIV", m_C02_canon_decode_err_EQUIV.Canonical, m_C02_canon_decode_err_EQUIV.CanonicalAsReference, m_C02_canon_decode_err_EQUIV.CanonicalForSigning},
	{"C10_wtm_key_notstring_UNREACHABLE", m_C10_wtm_key_notstring_UNREACHABLE.Canonical, m_C10_wtm_key_notstring_UNREACHABLE.CanonicalAsReference, m_C10_wtm_key_notstring_UNREACHABLE.CanonicalForSigning},
	{"C11_wtm_decode_err_EQUIV", m_C11_wtm_decode_err_EQUIV.Canonical, m_C11_wtm_decode_err_EQUIV.CanonicalAsReference, m_C11_wtm_decode_err_EQUIV.CanonicalForSigning},
	{"C22_snd_key_notstring_UNREACHABLE", m_C22_snd_key_notstring_UNREACHABLE.Canonical, m_C22_snd_key_notstring_UNREACHABLE.CanonicalAsReference, m_C22_snd_key_notstring_UNREACHABLE.CanonicalForSigning},
	{"C28_snd_default_UNREACHABLE", m_C28_snd_default_UNREACHABLE.Canonical, m_C28_snd_default_UNREACHABLE.CanonicalAsReference, m_C28_snd_default_UNREACHABLE.CanonicalForSigning},
	{"C47_wc_default_UNREACHABLE", m_C47_wc_default_UNREACHABLE.Canonical, m_C47_wc_default_UNREACHABLE.CanonicalAsReference, m_C47_wc_default_UNREACHABLE.CanonicalForSigning},
	{"C63_cn_first_empty_digits_EQUIV", m_C63_cn_first_empty_digits_EQUIV.Canonical, m_C63_cn_first_empty_digits_EQUIV.CanonicalAsReference, m_C63_cn_first_empty_digits_EQUIV.CanonicalForSigning},
	{"C64_cn_second_empty_digits_UNREACHABLE", m_C64_cn_second_empty_digits_UNREACHABLE.Canonical, m_C64_cn_second_empty_digits_UNREACHABLE.CanonicalAsReference, m_C64_cn_second_empty_digits_UNREACHABLE.CanonicalForSigning},
}

var frag = []string{"{", "}", "[", "]", `"a"`, `"signature"`, ":", ",", "1", "-0", "1e5", "1.5E-3", "true", "null", `"x\u00e9\n"`, " ", "\n", `"`, `\`, "x", "e", "0123", `"a":1`, `"signature":{}`, "0.0", "00", "-", "+", ".", "E", "9", "0"}

func gen(r *rand.Rand) string {
	n := r.Intn(14)
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteString(frag[r.Intn(len(frag))])
	}
	return b.String()
}
func num(r *rand.Rand) string {
	al := "0123456789.eE+-"
	b := make([]byte, 1+r.Intn(9))
	for i := range b {
		b[i] = al[r.Intn(len(al))]
	}
	return string(b)
}

func TestDiff(t *testing.T) {
	r := rand.New(rand.NewSource(11))
	docs := []string{""}
	for i := 0; i < 300000; i++ {
		docs = append(docs, gen(r))
	}
	for i := 0; i < 300000; i++ {
		x := num(r)
		docs = append(docs, x, "["+x+"]", `{"k":`+x+`}`)
	}
	al := "0123456789.eE+-"
	var rec func(p string, d int)
	rec = func(p string, d int) {
		if p != "" {
			docs = append(docs, p)
		}
		if d == 0 {
			return
		}
		for _, c := range al {
			rec(p+string(c), d-1)
		}
	}
	rec("", 5) // every numeric literal of up to 5 characters over the full number alphabet
	for _, d := range []int{9999, 10000, 10001, 20000} {
		docs = append(docs, strings.Repeat("[", d)+strings.Repeat("]", d))
	}
	diffs := map[string]int{}
	accepted := 0
	for _, d := range docs {
		b := []byte(d)
		if _, e := orig.Canonical(b); e == nil {
			accepted++
		}
		for _, m := range muts {
			if s(orig.Canonical(b)) != s(m.a(b)) {
				diffs[m.n]++
			}
			if s(orig.CanonicalAsReference(b)) != s(m.b(b)) {
				diffs[m.n]++
			}
			if s(orig.CanonicalForSigning(b)) != s(m.c(b)) {
				diffs[m.n]++
			}
		}
	}
	t.Logf("docs=%d accepted-by-original=%d", len(docs), accepted)
	for _, m := range muts {
		t.Logf("%s diffs=%d", m.n, diffs[m.n])
	}
}
