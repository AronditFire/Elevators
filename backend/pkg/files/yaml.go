package files

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
)

func DecodeYAML(r io.Reader, target any) error {
	d := yaml.NewDecoder(r)
	d.KnownFields(true)
	if err := d.Decode(target); err != nil {
		return fmt.Errorf("ошибка YAML: %w", err)
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return fmt.Errorf("ожидается один YAML-документ")
	}
	return nil
}

func EncodeYAML(value any) ([]byte, error) { return yaml.Marshal(value) }
