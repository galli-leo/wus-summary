package main

import (
	"os"
	"os/exec"
	"text/template"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func newLogger(name string) *zap.SugaredLogger {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.InitialFields = map[string]interface{}{
		"name": name,
	}
	logger, _ := config.Build()

	return logger.Sugar()
}

func CreateArray(args ...interface{}) []interface{} {
	return args
}

func main() {
	log := newLogger("main")
	log.Info("Starting Up")

	funcMap := template.FuncMap{
		"arr": CreateArray,
	}

	tmpl := template.New("tmpl.tex").Delims("[[", "]]").Funcs(funcMap)
	tmpl, err := tmpl.ParseFiles("tmpl.tex")
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}
	outf, _ := os.OpenFile("out.tex", os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	sections := GetDistrLinks()

	log.Info("Loading All Distribution Information")
	LoadAllDistrs(sections)
	log.Infof("Loaded All Distribution Information: %v", sections[0].Distrs[0])

	err = tmpl.Execute(outf, sections)
	if err != nil {
		log.Errorf("Failed to execute templates: %v", err)
	}

	outf.Close()

	cmd := exec.Command("pdflatex", "-interaction=nonstopmode", "-jobname=gen", "distr.tex")
	cmd.CombinedOutput()
	cmd2 := exec.Command("pdfjam", "--batch", "--nup", "4x1", "-o", "three.pdf", "--landscape", "gen.pdf")
	cmd2.Run()
}
