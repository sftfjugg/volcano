# Custom Plugin

## Background

Until now, plugins like `binpack`, `drf`, `gang` are provided by official.
But some users may want to implement the plugin by themself. So if scheduler can dynamicly load plugins, that will make it more flexible when handling different business scenarios.

## How to build a plugin

### 1. Coding

```go
// magic.go

package main  // note!!! package must be named main

import (
    "volcano.sh/volcano/pkg/scheduler/framework"
)

const PluginName = "magic"

type magicPlugin struct {}

func (mp *magicPlugin) Name() string {
    return PluginName
}

func New(arguments framework.Arguments) framework.Plugin {  // `New` is PluginBuilder
    return &magicPlugin{}
}

func (mp *magicPlugin) OnSessionOpen(ssn *framework.Session) {}

func (mp *magicPlugin) OnSessionClose(ssn *framework.Session) {}
```

### 2. Build the plugin to .so
```bash
GOARCH=amd64 go build -buildmode=plugin -o plugins/magic.so magic.go
```

### 3. Add plugins into container

Your can build your docker image

```dockerfile
FROM volcano.sh/vc-scheduler-pluginable:latest

COPY plugins plugins
```

Or just use `pvc` to mount these plugins

### 4. Specify deployment
```yaml
...
    containers:
    - name: volcano-scheduler
      image: volcano.sh/vc-scheduler-pluginable:latest
      args:
        - --logtostderr
        - --scheduler-conf=/volcano.scheduler/volcano-scheduler.conf
        - -v=3
        - --plugins-dir=plugins  # specify plugins dir path
        - 2>&1
```

## Note

1. Plugins should be rebuilded after volcano source code modified.
2. Plugin package name must be **main**.
