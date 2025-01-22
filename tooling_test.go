package simpleforce

import (
	"context"
	"testing"
)

func TestClient_Tooling_Query(t *testing.T) {
	ctx := context.Background()

	client := requireClient(ctx, t, true)

	q := "SELECT Id, Name FROM Layout WHERE Name = 'Account Layout'"
	result, err := client.Tooling().Query(ctx, q)
	if err != nil {
		t.FailNow()
	}
	if len(result.Records) > 0 {
		case0 := &result.Records[0]
		if case0.StringField("Name") != "Account Layout" {
			t.FailNow()
		}
	}
}

func TestClient_ExecuteAnonymous(t *testing.T) {
	ctx := context.Background()

	client := requireClient(ctx, t, true)

	apexBody := "System.debug('test');"
	result, err := client.ExecuteAnonymous(ctx, apexBody)
	if err != nil {
		t.FailNow()
	}
	if !result.Success {
		t.FailNow()
	}
}
