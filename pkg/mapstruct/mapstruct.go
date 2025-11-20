package mapstruct

import (
	"github.com/go-viper/mapstructure/v2"
)

func Decode(input any, output any) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Metadata: nil,
		TagName:  "json",
		Result:   output,
	})
	if err != nil {
		return err
	}

	return decoder.Decode(input)
}
