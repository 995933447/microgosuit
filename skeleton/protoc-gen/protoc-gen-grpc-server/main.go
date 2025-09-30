package main

import (
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"text/template"

	"flag"

	"github.com/995933447/microgosuit/skeleton"
	"github.com/995933447/microgosuit/skeleton/pb"
	"github.com/995933447/runtimeutil"
	"github.com/995933447/stringhelper-go"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/pluginpb"
)

func init() {
	// 自定义的 protoc 插件（例如 protoc-gen-xxx）必须通过 标准输入/输出 (stdin/stdout) 与 protoc 交互
	// 避免log 输出污染了 stdout，log 会把内容写到 stdout，而 protoc 会把 stdout 当成 CodeGeneratorResponse 解析。
	log.SetOutput(os.Stderr)
}

func main() {
	log.Println("======= Starting protoc-gen-grpc-server =========")

	if err := skeleton.LoadProtocGenConf(); err != nil {
		log.Fatal(runtimeutil.NewStackErr(err))
	}

	debug := flag.Bool("d", false, "是否开启debug")
	inputFile := flag.String("i", "", "调试pb")
	flag.Parse() // 解析命令行参数

	var (
		input []byte
		err   error
	)
	if *debug {
		if *inputFile == "" {
			log.Fatal("input file is required")
		}

		input, err = os.ReadFile("req.pb")
		if err != nil {
			log.Fatal(err)
		}
	} else {
		input, err = io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatal(runtimeutil.NewStackErr(err))
		}

		if skeleton.MustGetProtocGenConf().Debug {
			log.Println("enable debug, store input to a file: req.pb")
			err = os.WriteFile("./req.pb", input, os.ModePerm)
			if err != nil {
				log.Fatal(runtimeutil.NewStackErr(err))
			}
			return
		}
	}

	var req pluginpb.CodeGeneratorRequest
	if err := proto.Unmarshal(input, &req); err != nil {
		log.Fatal(runtimeutil.NewStackErr(err))
	}

	log.Println("Files to generate:", req.GetFileToGenerate())

	opts := protogen.Options{}
	plugin, err := opts.New(&req)
	if err != nil {
		log.Fatal(runtimeutil.NewStackErr(err))
	}

	for _, f := range plugin.Files {
		if !f.Generate {
			log.Printf("microgosuit gen-grpc-server, skipped gen %s\n", string(f.Desc.Name()))
			continue
		}

		//  只生成有 Service 的 proto
		if len(f.Services) == 0 {
			log.Printf("microgosuit gen-grpc-server, skipped gen %s\n", string(f.Desc.Name()))
			continue
		}

		if err = genServerSkeleton(plugin, f); err != nil {
			log.Fatal(runtimeutil.NewStackErr(err))
			return
		}
	}

	resp := &pluginpb.CodeGeneratorResponse{}
	out, err := proto.Marshal(resp)
	if err != nil {
		panic(err)
	}
	// 必须写到 stdout
	os.Stdout.Write(out)

	log.Printf("server generated successfully!\n")
}

func normalizeDirName(name string) string {
	switch skeleton.MustGetProtocGenConf().DirNamingMethod {
	case "camel":
		return stringhelper.LowerFirstASCII(stringhelper.Camel(name))
	case "snake":
		return stringhelper.Snake(name)
	default:
		return strings.Replace(stringhelper.Snake(name), "_", "", -1)
	}
}

type mainFileTemplateSlot struct {
	GrpcResolveSchema       string
	DiscoverPrefix          string
	ServiceNamespace        string
	ServiceImportPath       string
	ServiceServerImportPath string
	ServiceGoPackage        string
	ServiceNames            []string
	EnabledHealth           bool
}

type modNamesFileTemplateSlot struct {
	ServiceNamespace string
	ServiceNames     []string
}

type portFileTemplateSlot struct {
	RpcPort        string
	EnumImportPath string
}

type serviceHandlerFileTemplateSlot struct {
	ServiceName             string
	ServiceClientPackage    string
	ServiceClientImportPath string
}

type serviceHandlerMethodFileTemplateSlot struct {
	ServiceName string
	MethodName  string
	Imports     []string
	Req         string
	Resp        string
}

func genServerSkeleton(plugin *protogen.Plugin, f *protogen.File) error {
	projDir := skeleton.MustGetProtocGenConf().ProjectDir
	if projDir == "" {
		var err error
		projDir, err = os.Getwd()
		if err != nil {
			log.Println(runtimeutil.NewStackErr(err))
			return err
		}
		projDir = strings.TrimSuffix(projDir, "/")
	}

	log.Printf("Project dir: %s\n", projDir)

	goPkgName := normalizeDirName(string(f.GoImportPath + "Server"))
	svcRootDirPath := strings.TrimSuffix(projDir, "/") + "/" + goPkgName

	log.Printf("Service root dir path: %s\n", svcRootDirPath)

	if _, err := os.Stat(svcRootDirPath); err != nil {
		if !os.IsNotExist(err) {
			log.Println(runtimeutil.NewStackErr(err))
			return err
		}
		if err = os.MkdirAll(svcRootDirPath, os.ModePerm); err != nil {
			log.Println(runtimeutil.NewStackErr(err))
			return err
		}
	} else {
		log.Printf("Service root dir exists: %s\n", svcRootDirPath)
	}

	goModName := path.Dir(string(f.GoImportPath))
	projRootDirPath := strings.TrimSuffix(projDir, "/") + "/" + goModName

	log.Printf("Project root dir path: %s\n", projRootDirPath)

	// ========= go.mod =========
	goModFilePath := projRootDirPath + "/go.mod"
	if _, err := os.Stat(goModFilePath); err != nil {
		if !os.IsNotExist(err) {
			log.Println(runtimeutil.NewStackErr(err))
			return err
		}

		cmd := exec.Command("go", "mod", "init", goModName)
		cmd.Dir = projRootDirPath
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Println(runtimeutil.NewStackErr(err))
			return err
		}

		log.Print(string(output))
	} else {
		log.Printf("go mod file exists, skipped gen\n")
	}

	var serviceNames []string
	for _, service := range f.Services {
		serviceNames = append(serviceNames, service.GoName)
	}

	// ========= 生成 main.go =========
	svcMainFilePath := svcRootDirPath + "/main.go"
	if _, err := os.Stat(svcMainFilePath); err != nil {
		if !os.IsNotExist(err) {
			log.Println(runtimeutil.NewStackErr(err))
			return err
		}

		file, err := os.OpenFile(svcMainFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
		if err != nil {
			log.Println(runtimeutil.NewStackErr(err))
			return err
		}
		defer file.Close()

		tmpl, err := template.New("mainFileTemplate").Parse(mainFileTemplate)
		if err != nil {
			log.Println(runtimeutil.NewStackErr(err))
			return err
		}

		err = tmpl.Execute(file, &mainFileTemplateSlot{
			GrpcResolveSchema:       skeleton.MustGetProtocGenConf().GrpcResolveSchema,
			DiscoverPrefix:          skeleton.MustGetProtocGenConf().DiscoverPrefix,
			ServiceNamespace:        string(f.Desc.Package()),
			ServiceImportPath:       string(f.GoImportPath),
			ServiceServerImportPath: path.Dir(string(f.GoImportPath)) + "/" + normalizeDirName(string(f.GoPackageName+"Server")),
			ServiceGoPackage:        string(f.GoPackageName),
			ServiceNames:            serviceNames,
			EnabledHealth:           skeleton.MustGetProtocGenConf().EnabledHealth,
		})
		if err != nil {
			log.Println(runtimeutil.NewStackErr(err))
			return err
		}
	} else {
		log.Printf("%s exists. skipped gen\n", svcMainFilePath)
	}

	// ========= 生成 svc_name.go =========
	modNamesFilePath := svcRootDirPath + "/mod_name.go"
	modNamesFile, err := os.OpenFile(modNamesFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		log.Println(runtimeutil.NewStackErr(err))
		return err
	}
	defer modNamesFile.Close()

	tmpl, err := template.New("modNamesFileTemplate").Parse(modNamesFileTemplate)
	if err != nil {
		log.Println(runtimeutil.NewStackErr(err))
		return err
	}

	err = tmpl.Execute(modNamesFile, &modNamesFileTemplateSlot{
		ServiceNamespace: string(f.GoPackageName),
		ServiceNames:     serviceNames,
	})
	if err != nil {
		log.Println(runtimeutil.NewStackErr(err))
		return err
	}

	// ========= 查找 RpcPort =========
	var (
		portEnumImportPath string
		port               int32
		enumKey            = "RpcPort" + stringhelper.UpperFirstASCII(stringhelper.Camel(string(f.Desc.Package())))
	)
	for _, enum := range f.Enums {
		ext := proto.GetExtension(enum.Desc.Options(), pb.E_IsRpcPort)
		if isRpcPort, ok := ext.(bool); !ok || !isRpcPort {
			continue
		}
		portEnumImportPath = string(f.GoImportPath)
		for _, value := range enum.Values {
			if string(value.Desc.Name()) == enumKey {
				port = int32(value.Desc.Number())
				log.Println("find rpc port in top level enum")
				break
			}
		}
	}

	if port == 0 {
		// imports 的枚举
		imports := f.Desc.Imports()
		for i := 0; i < imports.Len(); i++ {
			imp := imports.Get(i)
			importedFile := imp.FileDescriptor

			enums := importedFile.Enums()
			for j := 0; j < enums.Len(); j++ {
				enum := enums.Get(j)
				ext := proto.GetExtension(enum.Options(), pb.E_IsRpcPort)
				if isRpcPort, ok := ext.(bool); !ok || !isRpcPort {
					continue
				}
				portEnumImportFile := plugin.FilesByPath[imp.Path()]
				portEnumImportPath = string(portEnumImportFile.GoImportPath)
				values := enum.Values()
				for k := 0; k < values.Len(); k++ {
					val := values.Get(k)
					if string(val.Name()) == enumKey {
						port = int32(val.Number())
						break
					}
				}
			}
		}
	}

	log.Println("portEnumImportPath", portEnumImportPath)
	if port > 0 {
		log.Printf("specify rpc port: %d\n", port)
	} else {
		log.Printf("specify no rpc port\n")
	}

	// ========= 生成 net.go =========
	portFilePath := svcRootDirPath + "/net.go"
	portFile, err := os.OpenFile(portFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		log.Println(runtimeutil.NewStackErr(err))
		return err
	}
	defer portFile.Close()

	tmpl, err = template.New("portFileTemplate").Parse(portFileTemplate)
	if err != nil {
		log.Println(runtimeutil.NewStackErr(err))
		return err
	}

	var portStr string
	if port != 0 {
		portStr = "int(" + path.Base(portEnumImportPath) + "." + "RpcPort_" + enumKey + ")"
	} else {
		portStr = "0"
	}
	err = tmpl.Execute(portFile, &portFileTemplateSlot{
		RpcPort:        portStr,
		EnumImportPath: portEnumImportPath,
	})
	if err != nil {
		log.Println(runtimeutil.NewStackErr(err))
		return err
	}

	// ========= 生成 boot.go =========
	bootDirPath := svcRootDirPath + "/" + "boot"
	if _, err = os.Stat(bootDirPath); err != nil {
		if !os.IsNotExist(err) {
			log.Println(runtimeutil.NewStackErr(err))
			return err
		}

		if err = os.MkdirAll(bootDirPath, os.ModePerm); err != nil {
			log.Println(runtimeutil.NewStackErr(err))
			return err
		}
	}

	bootFilePath := bootDirPath + "/boot.go"

	if _, err = os.Stat(bootFilePath); err != nil {
		bootFile, err := os.OpenFile(bootFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
		if err != nil {
			log.Println(runtimeutil.NewStackErr(err))
			return err
		}
		defer bootFile.Close()

		tmpl, err = template.New("bootFileTemplate").Parse(bootFileTemplate)
		if err != nil {
			log.Println(runtimeutil.NewStackErr(err))
			return err
		}

		err = tmpl.Execute(bootFile, nil)
		if err != nil {
			log.Println(runtimeutil.NewStackErr(err))
			return err
		}
	} else {
		log.Printf("%s exists, skipped gen\n", bootFilePath)
	}

	// ========= 生成 Handler =========
	for _, service := range f.Services {
		svcHandlerDirPath := svcRootDirPath + "/handler"
		if _, err := os.Stat(svcHandlerDirPath); err != nil {
			if !os.IsNotExist(err) {
				log.Println(runtimeutil.NewStackErr(err))
				return err
			}
			if err = os.MkdirAll(svcHandlerDirPath, os.ModePerm); err != nil {
				log.Println(runtimeutil.NewStackErr(err))
				return err
			}
		}

		err := func() error {
			svcHandlerFilePath := svcHandlerDirPath + "/" + stringhelper.Snake(service.GoName) + "_hdl.go"
			if _, err := os.Stat(svcHandlerFilePath); os.IsNotExist(err) {
				file, err := os.OpenFile(svcHandlerFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
				if err != nil {
					log.Println(runtimeutil.NewStackErr(err))
					return err
				}
				defer file.Close()

				tmpl, err := template.New("svcHandlerTmpl").Parse(serviceHandlerFileTemplate)
				if err != nil {
					log.Println(runtimeutil.NewStackErr(err))
					return err
				}

				err = tmpl.Execute(file, &serviceHandlerFileTemplateSlot{
					ServiceName:             service.GoName,
					ServiceClientImportPath: string(f.GoImportPath),
					ServiceClientPackage:    string(f.GoPackageName),
				})
				if err != nil {
					log.Println(runtimeutil.NewStackErr(err))
					return err
				}
			} else {
				log.Printf("%s exists, skipped gen\n", svcHandlerFilePath)
			}

			// ========= 生成方法 =========
			for _, m := range service.Methods {
				svcHandlerMethodFilePath := svcHandlerDirPath + "/" +
					stringhelper.Snake(service.GoName) + "_hdl_" + stringhelper.Snake(m.GoName) + ".go"

				if _, err := os.Stat(svcHandlerMethodFilePath); err != nil {
					if !os.IsNotExist(err) {
						log.Println(runtimeutil.NewStackErr(err))
						return err
					}

					err = func() error {
						svcHandlerMethodFile, err := os.OpenFile(svcHandlerMethodFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
						if err != nil {
							log.Println(runtimeutil.NewStackErr(err))
							return err
						}
						defer svcHandlerMethodFile.Close()

						var imports []string
						if !m.Desc.IsStreamingServer() && !m.Desc.IsStreamingClient() {
							tmpl, err = template.New("serviceHandlerUnaryMethodFileTemplate").Parse(serviceHandlerUnaryMethodFileTemplate)
							if err != nil {
								log.Println(runtimeutil.NewStackErr(err))
								return err
							}
							imports = append(imports, "context")
						} else if m.Desc.IsStreamingServer() && !m.Desc.IsStreamingClient() {
							tmpl, err = template.New("serviceHandlerServerStreamMethodFileTemplate").Parse(serviceHandlerServerStreamMethodFileTemplate)
							if err != nil {
								log.Println(runtimeutil.NewStackErr(err))
								return err
							}
							imports = append(imports, "google.golang.org/grpc")
						} else if m.Desc.IsStreamingClient() && !m.Desc.IsStreamingServer() {
							tmpl, err = template.New("serviceHandlerClientStreamMethodFileTemplate").Parse(serviceHandlerClientStreamMethodFileTemplate)
							if err != nil {
								log.Println(runtimeutil.NewStackErr(err))
								return err
							}
							imports = append(imports, "google.golang.org/grpc")
						} else {
							tmpl, err = template.New("serviceHandlerBothStreamMethodFileTemplate").Parse(serviceHandlerBothStreamMethodFileTemplate)
							if err != nil {
								log.Println(runtimeutil.NewStackErr(err))
								return err
							}
							imports = append(imports, "google.golang.org/grpc")
						}

						//  取 import & 类型名
						reqImport := string(m.Input.GoIdent.GoImportPath)
						respImport := string(m.Output.GoIdent.GoImportPath)
						reqPkg := path.Base(reqImport)
						respPkg := path.Base(respImport)

						imports = append(imports, reqImport)
						if reqImport != respImport {
							imports = append(imports, respImport)
						}
						reqName := reqPkg + "." + m.Input.GoIdent.GoName
						respName := respPkg + "." + m.Output.GoIdent.GoName

						err = tmpl.Execute(svcHandlerMethodFile, &serviceHandlerMethodFileTemplateSlot{
							ServiceName: service.GoName,
							MethodName:  m.GoName,
							Req:         reqName,
							Resp:        respName,
							Imports:     imports,
						})
						if err != nil {
							log.Println(runtimeutil.NewStackErr(err))
							return err
						}
						return nil
					}()
					if err != nil {
						return err
					}
				} else {
					log.Printf("%s exists, skipped gen\n", svcHandlerMethodFilePath)
				}
			}
			return nil
		}()
		if err != nil {
			return err
		}
	}

	return nil
}
