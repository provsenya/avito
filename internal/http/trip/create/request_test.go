package create

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecode(t *testing.T) {
	valid := `{"user_id":"5cb72c04-7650-45c9-a79b-bcdba0631e0c","driver_id":"8860b315-ec86-42eb-a17c-7c163d721ff5","start_point":{"latitude":0,"longitude":0},"end_point":{"latitude":0,"longitude":0},"price":0}`
	for _, tc := range []struct {
		name, body string
		ok         bool
	}{
		{"valid", valid, true}, {"missing price", strings.Replace(valid, `,"price":0`, "", 1), false}, {"null", strings.Replace(valid, `"price":0`, `"price":null`, 1), false}, {"missing latitude", strings.Replace(valid, `"latitude":0,`, "", 1), false}, {"unknown", strings.Replace(valid, `"price":0`, `"price":0,"extra":1`, 1), false}, {"multiple", valid + valid, false}, {"malformed", `{`, false}, {"empty", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/", strings.NewReader(tc.body))
			r.Header.Set("Content-Type", "application/json")
			_, err := decodeCreateTripRequest(httptest.NewRecorder(), r)
			if (err == nil) != tc.ok {
				t.Fatalf("error=%v", err)
			}
		})
	}
}
