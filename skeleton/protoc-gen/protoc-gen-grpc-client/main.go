package main

import (
	"bytes"
	"flag"
	"io"
	"log"
	"os"
	"text/template"

	"github.com/995933447/microgosuit/skeleton"
	"github.com/995933447/runtimeutil"
	"github.com/995933447/stringhelper-go"
	"github.com/golang/protobuf/proto"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"
)

func init() {
	// 自定义的 protoc 插件（例如 protoc-gen-xxx）必须通过 标准输入/输出 (stdin/stdout) 与 protoc 交互
	// 避免log 输出污染了 stdout，log 会把内容写到 stdout，而 protoc 会把 stdout 当成 CodeGeneratorResponse 解析。
	log.SetOutput(os.Stderr)
}

func main() {
	log.Println("======= Starting protoc-gen-grpc-client =========")

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
			log.Printf("microgosuit gen-grpc-client, skipped gen %s\n", string(f.Desc.Name()))
			continue
		}

		//  只生成有 Service 的 proto
		if len(f.Services) == 0 {
			log.Printf("microgosuit gen-grpc-client, skipped gen %s\n", string(f.Desc.Name()))
			continue
		}

		if err = genClientSkeleton(plugin, f); err != nil {
			log.Fatal(runtimeutil.NewStackErr(err))
		}
	}

	stdout := plugin.Response()
	out, err := proto.Marshal(stdout)
	if err != nil {
		panic(err)
	}

	// 必须写到 stdout
	os.Stdout.Write(out)

	log.Printf("client generated successfully!\n")
}

type rpcFileHeadTemplateSlot struct {
	ServiceNamespace    string
	ShouldImportContext bool
}

type rpcFileDefineServiceTemplateSlot struct {
	ServiceNamespace      string
	ServiceName           string
	ServiceNameLowerCamel string
	ResolveSchema         string
}

type rpcFileServiceMethodTemplateSlot struct {
	ServiceName string
	MethodName  string
	Req         string
	Resp        string
}

func genClientSkeleton(plugin *protogen.Plugin, f *protogen.File) error {
	var b bytes.Buffer

	tmpl, err := template.New("rpcFileHeadTemplate").Parse(rpcFileHeadTemplate)
	if err != nil {
		log.Println(runtimeutil.NewStackErr(err))
		return err
	}

	var shouldImportContext bool
	for _, service := range f.Services {
		if len(service.Methods) == 0 {
			continue
		}
		shouldImportContext = true
		break
	}

	var bb bytes.Buffer
	err = tmpl.Execute(&bb, &rpcFileHeadTemplateSlot{
		ServiceNamespace:    string(f.Desc.Package()),
		ShouldImportContext: shouldImportContext,
	})
	if err != nil {
		log.Println(runtimeutil.NewStackErr(err))
		return err
	}

	b.Write(bb.Bytes())

	for _, service := range f.Services {
		bb.Reset()

		tmpl, err = template.New("rpcFileDefineServiceTemplate").Parse(rpcFileDefineServiceTemplate)
		if err != nil {
			log.Println(runtimeutil.NewStackErr(err))
			return err
		}

		svcNameLowerCamp := stringhelper.Camel(service.GoName)
		err = tmpl.Execute(&bb, &rpcFileDefineServiceTemplateSlot{
			ServiceName:           service.GoName,
			ServiceNameLowerCamel: svcNameLowerCamp,
			ResolveSchema:         skeleton.MustGetProtocGenConf().GrpcResolveSchema,
			ServiceNamespace:      string(f.Desc.Package()),
		})
		if err != nil {
			log.Println(runtimeutil.NewStackErr(err))
			return err
		}

		b.Write(bb.Bytes())

		for _, method := range service.Methods {
			bb.Reset()
			if !method.Desc.IsStreamingClient() && !method.Desc.IsStreamingServer() {
				tmpl, err = template.New("rpcFileServiceUnaryMethodTemplate").Parse(rpcFileServiceUnaryMethodTemplate)
				if err != nil {
					log.Println(runtimeutil.NewStackErr(err))
					return err
				}
			} else if method.Desc.IsStreamingClient() && !method.Desc.IsStreamingServer() {
				tmpl, err = template.New("rpcFileServiceClientStreamMethodTemplate").Parse(rpcFileServiceClientStreamMethodTemplate)
				if err != nil {
					log.Println(runtimeutil.NewStackErr(err))
					return err
				}
			} else if method.Desc.IsStreamingServer() && !method.Desc.IsStreamingClient() {
				tmpl, err = template.New("rpcFileServiceServerStreamMethodTemplate").Parse(rpcFileServiceServerStreamMethodTemplate)
				if err != nil {
					log.Println(runtimeutil.NewStackErr(err))
					return err
				}
			} else {
				tmpl, err = template.New("rpcFileServiceBothStreamMethodTemplate").Parse(rpcFileServiceBothStreamMethodTemplate)
				if err != nil {
					log.Println(runtimeutil.NewStackErr(err))
					return err
				}
			}

			err = tmpl.Execute(&bb, &rpcFileServiceMethodTemplateSlot{
				ServiceName: service.GoName,
				MethodName:  method.GoName,
				Req:         method.Input.GoIdent.GoName,
				Resp:        method.Output.GoIdent.GoName,
			})
			if err != nil {
				log.Println(runtimeutil.NewStackErr(err))
				return err
			}

			b.Write(bb.Bytes())
		}
	}

	if _, err = plugin.NewGeneratedFile(f.GeneratedFilenamePrefix+"_microgosuit.pb.go", ".").Write(b.Bytes()); err != nil {
		log.Println(runtimeutil.NewStackErr(err))
		return err
	}

	return nil
}
