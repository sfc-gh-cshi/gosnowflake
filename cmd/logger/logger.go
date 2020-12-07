package main

import (
	"bytes"
	rlog "github.com/sirupsen/logrus"
	sf "github.com/snowflakedb/gosnowflake"
	"log"
	"strings"
)

type testLogger struct {
	rlog.Logger
}

func getLogger() testLogger {
	var logging = testLogger{*rlog.New()}
	var formatter = rlog.JSONFormatter{CallerPrettyfier: sf.SFCallerPrettyfier}
	logging.SetReportCaller(true)
	logging.SetFormatter(&formatter)
	return logging
}

func main() {
	buf := &bytes.Buffer{}
	buf2 := &bytes.Buffer{}

	sf.GetLogger().SetOutput(buf)
	sf.GetLogger().Info("Hello I am default")
	sf.GetLogger().Info("Hello II amm default")

	var testlg = getLogger()
	testlg.SetOutput(buf2)
	sf.SetLogger(&testlg)
	sf.GetLogger().Info("Hello I am new")

	log.Print("Expect all true values:")
	log.Printf("%t:%t:%t:%t", strings.Contains(buf.String(), "I am default"),
		strings.Contains(buf.String(), "II amm default"),
		!strings.Contains(buf.String(), "I am new"),
		strings.Contains(buf2.String(), "I am new"))

}
