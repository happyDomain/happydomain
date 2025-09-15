PLUGIN_NAME=matrix
TARGET=../happydomain-plugin-test-$(PLUGIN_NAME).so

all: $(TARGET)

$(TARGET): *.go
	go build -buildmode=plugin -o $@ git.happydns.org/happyDomain/plugins/$(PLUGIN_NAME)
