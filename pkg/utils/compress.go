package utils

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
)

func TarGzDirectory(dir string) (io.Reader, error) {
	buf := bytes.Buffer{}
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	if err := tw.AddFS(os.DirFS(dir)); err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gw.Close(); err != nil {
		return nil, err
	}
	return &buf, nil
}
