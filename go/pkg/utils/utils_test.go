package utils

import (
	"os"
	"testing"
)

func TestEnv(t *testing.T) {

	os.Setenv("SERVER_HOST", "test")
	os.Setenv("GQL_SERVER_GRAPHQL_COMPLEXITY_LIMIT", "10")
	os.Setenv("GQL_SERVER_GRAPHQL_PLAYGROUND_ENABLED", "true")

	want := "test"
	got := GetStringEnv("SERVER_HOST", "else")

	if got != want {
		t.Errorf("TestEnv = %q, want %q", got, want)
	}

	wantInt := 10
	gotInt := GetIntEnv("GQL_SERVER_GRAPHQL_COMPLEXITY_LIMIT", 5)

	if gotInt != wantInt {
		t.Errorf("TestEnv = %d, want %d", gotInt, wantInt)
	}

	wantInt64 := int64(10)
	gotInt64 := GetInt64Env("GQL_SERVER_GRAPHQL_COMPLEXITY_LIMIT", int64(5))

	if gotInt != wantInt {
		t.Errorf("TestEnv = %d, want %d", gotInt64, wantInt64)
	}

	wantBool := true
	gotBool := GetBoolEnv("GQL_SERVER_GRAPHQL_PLAYGROUND_ENABLED", false)

	if gotBool != wantBool {
		t.Errorf("TestEnv = %t, want %t", gotBool, wantBool)
	}
}
