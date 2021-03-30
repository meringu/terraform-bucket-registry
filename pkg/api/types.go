package api

type ProviderPackage struct {
	Protocols           []string     `json:"protocols"`
	OS                  string       `json:"os"`
	Arch                string       `json:"arch"`
	Filename            string       `json:"filename"`
	DownloadURL         string       `json:"download_url"`
	ShasumsURL          string       `json:"shasums_url"`
	ShasumsSignatureURL string       `json:"shasums_signature_url"`
	Shasum              string       `json:"shasum"`
	SigningKeys         *SigningKeys `json:"signing_keys"`
}

type SigningKeys struct {
	GPGPublicKeys []*GPGPublicKey `json:"gpg_public_keys"`
}

type GPGPublicKey struct {
	KeyID          string `json:"key_id"`
	ASCIIArmor     string `json:"ascii_armor"`
	TrustSignature string `json:"trust_signature"`
	Source         string `json:"source"`
	SourceURL      string `json:"source_url"`
}

type ProviderVersions struct {
	Versions []*ProviderVersion `json:"versions"`
}

type ProviderVersion struct {
	Version   string      `json:"version"`
	Protocols []string    `json:"protocols"`
	Platforms []*Platform `json:"platform"`
}

type Platform struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}
