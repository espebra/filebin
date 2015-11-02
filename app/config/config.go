package config

type Configuration struct {
	Host                string
	Port                int
	Readtimeout         int
	Writetimeout        int
	Maxheaderbytes      int
	Filedir             string
	Baseurl             string
	Tempdir             string
	Logdir              string
	//Thumbdir            string
	//GeoIP2              string
	Verbose             bool
	//Database            string
	//Pagination          int
	TriggerNewTag       string
	TriggerUploadedFile string
	TriggerExpiredTag   string
}

var Global Configuration

func init() {
	Global = Configuration{
		Host: "127.0.0.1",
		Port: 31337,
		Readtimeout: 3600,
		Writetimeout: 3600,
		Maxheaderbytes: 1 << 20,
		Baseurl: "http://localhost:31337",
		Filedir: "/srv/filebin/files",
		Tempdir: "/srv/filebin/temp",
		Logdir: "/var/log/filebin",
		//Thumbdir: "/srv/filebin/thumbnails",
		//Database: "/srv/filebin/filebin.db",
		//GeoIP2: "/srv/filebin/GeoLite2-Country.mmdb",
		//Pagination: 120,
		TriggerNewTag: "",
		TriggerUploadedFile: "",
		TriggerExpiredTag: "",
	}
}

