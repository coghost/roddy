package roddy

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type UtilSuite struct {
	suite.Suite
}

func TestUtil(t *testing.T) {
	suite.Run(t, new(UtilSuite))
}

func (s *UtilSuite) SetupSuite() {
}

func (s *UtilSuite) TearDownSuite() {
}

func (s *UtilSuite) Test_00_FilenameFromURL() {
	tests := []struct {
		uri   string
		wantS string
		wantE error
	}{
		{"https://github.com/coghost/roddy", "https_github.com", nil},
		{"https://go-rod.github.io/i18n/zh-CN/#/browsers-pages", "https_go-rod.github.io", nil},
		{
			"https://bryanftan.medium.com/accept-interfaces-return-structs-in-go-d4cab29a301b",
			"https_bryanftan.medium.com",
			nil,
		},
	}
	for _, tt := range tests {
		v, e := FilenameFromUrl(tt.uri)
		s.Equal(tt.wantS, v)
		s.Equal(tt.wantE, e)
	}
}
