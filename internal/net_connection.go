package internal

import (
	"example/hello/message"
	"fmt"
	"github.com/mitchellh/mapstructure"
)

type NetConnectionConnectCommand struct {
	App            string               `mapstructure:"app" amf0:"app"`
	Type           string               `mapstructure:"type" amf0:"type"`
	FlashVer       string               `mapstructure:"flashVer" amf0:"flashVer"`
	TCURL          string               `mapstructure:"tcUrl" amf0:"tcUrl"`
	Fpad           bool                 `mapstructure:"fpad" amf0:"fpad"`
	Capabilities   int                  `mapstructure:"capabilities" amf0:"capabilities"`
	AudioCodecs    int                  `mapstructure:"audioCodecs" amf0:"audioCodecs"`
	VideoCodecs    int                  `mapstructure:"videoCodecs" amf0:"videoCodecs"`
	VideoFunction  int                  `mapstructure:"videoFunction" amf0:"videoFunction"`
	ObjectEncoding message.EncodingType `mapstructure:"objectEncoding" amf0:"objectEncoding"`
}
type NetConnectionConnect struct {
	Command NetConnectionConnectCommand
}

func (t *NetConnectionConnect) FromArgs(args ...interface{}) error {
	command := args[0].(map[string]interface{})
	// map에 저장된 데이터를 구조체로 변환
	if err := mapstructure.Decode(command, &t.Command); err != nil {
		return fmt.Errorf("Failed to mapping NetConnectionConnect")
	}

	return nil
}

func (t *NetConnectionConnect) ToArgs(ty message.EncodingType) ([]interface{}, error) {
	return []interface{}{
		t.Command,
	}, nil
}

type NetConnectionReleaseStream struct {
	StreamName string
}

func (t *NetConnectionReleaseStream) FromArgs(args ...interface{}) error {
	// args[0] is unknown, ignore
	t.StreamName = args[1].(string)

	return nil
}

func (t *NetConnectionReleaseStream) ToArgs(ty message.EncodingType) ([]interface{}, error) {
	return []interface{}{
		nil, // no command object
		t.StreamName,
	}, nil
}
