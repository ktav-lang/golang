package engine

import (
	"math/big"
)

// New pure insertions must fit the numeric domain the source parser can
// represent: Integer is signed i64, while canonical decimal tokens outside
// that range are strings. Existing values are not rewritten by this check.
func validateEditIntegerDomain(v *pvalue) *Error {
	switch v.kind {
	case kInt:
		n, ok := new(big.Int).SetString(v.s, 10)
		if !ok || !n.IsInt64() {
			return editUnsupported("inserted integer is outside the signed 64-bit range")
		}
	case kArr:
		for _, child := range v.arr {
			if err := validateEditIntegerDomain(child); err != nil {
				return err
			}
		}
	case kObj:
		for _, e := range v.obj.entries {
			if err := validateEditIntegerDomain(e.val); err != nil {
				return err
			}
		}
	}
	return nil
}
