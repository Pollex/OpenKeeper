package oas3_test

import (
	"fmt"
	"openkeeper/transformers/oas3"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestABC(t *testing.T) {
	b, err := os.Open("../../test/petstore.json")
	require.NoError(t, err)
	rules, err := oas3.FromStream(oas3.Config{}, b)
	require.NoError(t, err)
	require.NotNil(t, rules)
	fmt.Printf("rules: %v\n", rules)
	t.FailNow()
}
