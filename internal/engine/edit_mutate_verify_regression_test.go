package engine

import "testing"

func TestVerifyMutatedRejectsDifferentContent(t *testing.T) {
	expected := objectWithEntries(oentry{"a", pvInt("1")})
	if err := verifyMutated("a: 2\n", expected); err == nil {
		t.Fatal("verifyMutated accepted different content")
	}
}

func TestVerifyMutatedKeyOrderRejectsDifferentOrder(t *testing.T) {
	expected := objectWithEntries(oentry{"a", pvInt("1")}, oentry{"b", pvInt("2")})
	if err := verifyMutatedKeyOrder("b: 2\na: 1\n", expected); err == nil {
		t.Fatal("verifyMutatedKeyOrder accepted different key order")
	}
}
