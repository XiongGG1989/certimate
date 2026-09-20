package huaweicloudwaf

import "testing"

func TestNormalizePEMNewlines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "LF",
			input: "-----BEGIN PRIVATE KEY-----\nkey-data\n-----END PRIVATE KEY-----\n",
			want:  "-----BEGIN PRIVATE KEY-----\nkey-data\n-----END PRIVATE KEY-----\n",
		},
		{
			name:  "CRLF",
			input: "-----BEGIN CERTIFICATE-----\r\ncert-data\r\n-----END CERTIFICATE-----\r\n",
			want:  "-----BEGIN CERTIFICATE-----\ncert-data\n-----END CERTIFICATE-----\n",
		},
		{
			name:  "CR",
			input: "line-1\rline-2",
			want:  "line-1\nline-2",
		},
		{
			name:  "NoNewlines",
			input: "key-data",
			want:  "key-data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizePEMNewlines(tt.input); got != tt.want {
				t.Fatalf("normalizePEMNewlines() = %q, want %q", got, tt.want)
			}
		})
	}
}
