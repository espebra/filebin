package config

type filters []string

func (i *filters) String() string {
	return "string representation"
}

func (i *filters) Set(value string) error {
	*i = append(*i, value)
	return nil
}

type Configuration struct {
	Host                string
	Port                int
	ReadTimeout         int
	WriteTimeout        int
	MaxHeaderBytes      int
	Filedir             string
	Baseurl             string
	Tempdir             string
	Expiration          int64
	TriggerNewBin       string
	TriggerUploadFile   string
	TriggerDownloadBin  string
	TriggerDownloadFile string
	TriggerDeleteBin    string
	TriggerDeleteFile   string
	TriggerExpireBin    string
	ClientAddrHeader    string
	DefaultBinLength    int
	Workers             int
	Version             bool
	CacheInvalidation   bool
	AdminUsername       string
	AdminPassword       string
	AccessLog           string
	Filters             filters
	HotLinking          bool
}

var Global Configuration

func init() {
	Global = Configuration{
		Host:           "127.0.0.1",
		Port:           31337,
		ReadTimeout:    3600,
		WriteTimeout:   3600,
		MaxHeaderBytes: 1 << 20,
		// 7776000 = 3 months
		Expiration:        7776000,
		Baseurl:           "http://localhost:31337",
		Filedir:           "/srv/filebin/files",
		Tempdir:           "/tmp",
		DefaultBinLength:  16,
		Workers:           1,
		CacheInvalidation: false,
		AdminUsername:     "admin",
		ClientAddrHeader:  "",
		AccessLog:         "/var/log/filebin/access.log",
		Filters:           []string{},
		HotLinking:        true,
	}
}
