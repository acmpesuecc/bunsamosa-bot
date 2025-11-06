package handlers

import "testing"

type parseBountyTest struct {
	comment string
	valid   bool
	bounty  int
}

var parseBountyTests = []parseBountyTest{
	{comment: "!bounty 20", valid: true, bounty: 20},
	{comment: " !bounty 20", valid: true, bounty: 20},
	{comment: " !bounty 20\n", valid: true, bounty: 20},
	{comment: "!bounty 30 @appy", valid: true, bounty: 30},
	{comment: "!bounty", valid: false, bounty: -1},
	{comment: "!bounty ", valid: false, bounty: -1},
	{comment: "!bounty abcd", valid: false, bounty: -1},
}

func TestParseBountyPoints(t *testing.T) {
	for _, test := range parseBountyTests {
		bounty, valid := parseBountyPoints(test.comment)
		if test.valid {
			if !(valid && bounty == test.bounty) {
				t.Errorf("Expected valid=true, Got valid=%v\nExpected bounty=%d, Got bounty=%d", test.valid, test.bounty, bounty)
			}
		} else {
			if valid {
				t.Errorf("Expected valid=false, Got valid=%v\nExpected bounty=%d, Got bounty=%d", test.valid, test.bounty, bounty)
			}
		}
	}
}
