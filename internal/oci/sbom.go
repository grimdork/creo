package oci

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

type spdxDocument struct {
	SPDXVersion   string        `json:"spdxVersion"`
	DataLicense   string        `json:"dataLicense"`
	SPDXID        string        `json:"SPDXID"`
	Name          string        `json:"name"`
	CreationInfo  spdxCreation  `json:"creationInfo"`
	Packages      []spdxPackage `json:"packages"`
	Files         []spdxFile    `json:"files"`
	Relationships []spdxRel     `json:"relationships"`
}

type spdxCreation struct {
	Created  string   `json:"created"`
	Creators []string `json:"creators"`
}

type spdxPackage struct {
	Name             string   `json:"name"`
	VersionInfo      string   `json:"versionInfo"`
	SPDXID           string   `json:"SPDXID"`
	DownloadLocation string   `json:"downloadLocation"`
	FilesAnalyzed    bool     `json:"filesAnalyzed"`
	HasFiles         []string `json:"hasFiles"`
	LicenseConcluded string   `json:"licenseConcluded"`
	LicenseDeclared  string   `json:"licenseDeclared"`
	CopyrightText    string   `json:"copyrightText"`
}

type spdxFile struct {
	FileName         string         `json:"fileName"`
	SPDXID           string         `json:"SPDXID"`
	Checksums        []spdxChecksum `json:"checksums"`
	LicenseConcluded string         `json:"licenseConcluded"`
	CopyrightText    string         `json:"copyrightText"`
}

type spdxChecksum struct {
	Algorithm string `json:"algorithm"`
	Value     string `json:"value"`
}

type spdxRel struct {
	SPDXElementID    string `json:"spdxElementId"`
	RelatedElement   string `json:"relatedSpdxElement"`
	RelationshipType string `json:"relationshipType"`
}

func generateSBOM(binaryPath, name string) ([]byte, error) {
	data, err := os.ReadFile(binaryPath)
	if err != nil {
		return nil, err
	}

	h := sha256.Sum256(data)

	version := "unknown"
	info, err := buildinfo.Read(bytes.NewReader(data))
	if err == nil && info.Main.Version != "" {
		version = info.Main.Version
	}

	doc := spdxDocument{
		SPDXVersion: "SPDX-2.3",
		DataLicense: "CC0-1.0",
		SPDXID:      "SPDXRef-DOCUMENT",
		Name:        name + "-" + version,
		CreationInfo: spdxCreation{
			Created:  "1970-01-01T00:00:00Z",
			Creators: []string{"Tool: creo"},
		},
		Packages: []spdxPackage{
			{
				Name:             name,
				VersionInfo:      version,
				SPDXID:           "SPDXRef-Package",
				DownloadLocation: "NOASSERTION",
				FilesAnalyzed:    true,
				HasFiles:         []string{"SPDXRef-File-binary"},
				LicenseConcluded: "NOASSERTION",
				LicenseDeclared:  "NOASSERTION",
				CopyrightText:    "NOASSERTION",
			},
		},
		Files: []spdxFile{
			{
				FileName:         filepath.Join("/app", name),
				SPDXID:           "SPDXRef-File-binary",
				Checksums:        []spdxChecksum{{Algorithm: "SHA256", Value: hex.EncodeToString(h[:])}},
				LicenseConcluded: "NOASSERTION",
				CopyrightText:    "NOASSERTION",
			},
		},
		Relationships: []spdxRel{
			{
				SPDXElementID:    "SPDXRef-DOCUMENT",
				RelatedElement:   "SPDXRef-Package",
				RelationshipType: "DESCRIBES",
			},
			{
				SPDXElementID:    "SPDXRef-Package",
				RelatedElement:   "SPDXRef-File-binary",
				RelationshipType: "CONTAINS",
			},
		},
	}

	return json.MarshalIndent(doc, "", "  ")
}

func sbomLayer(data []byte) (v1.Layer, error) {
	return layerFromBytes("sbom.spdx.json", data, 0644)
}

func layerFromBytes(path string, data []byte, mode int64) (v1.Layer, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	if err := tw.WriteHeader(&tar.Header{
		Name:     path,
		Size:     int64(len(data)),
		Mode:     mode,
		ModTime:  time.Time{},
		Typeflag: tar.TypeReg,
	}); err != nil {
		return nil, err
	}
	if _, err := tw.Write(data); err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}

	return tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(buf.Bytes())), nil
	})
}
