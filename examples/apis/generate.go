package apis

//go:generate xrd-gen paths=./... xrd:allowDangerousTypes=true,crdVersions=v1 object:headerFile=../hack/boilerplate.go.txt,year=2024 output:artifacts:config=../package/xrds
