package registry

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"strings"

	protoparser "github.com/yoheimuta/go-protoparser/v4"
	protoparserparser "github.com/yoheimuta/go-protoparser/v4/parser"
)

// getAllProtoInfo uses DFS to fetch all the files from all directories passed and stores relevant proto files
func (r *Registry) getAllProtoInfo(protoFile string) ([]string, error) {
	visited := make(map[string]struct{}) // to make sure we don't end up in a loop
	result := make([]string, 0)

	var dfs func(protoFile string) error
	dfs = func(protoFile string) error {
		if _, ok := visited[protoFile]; ok {
			return nil
		}
		visited[protoFile] = struct{}{}
		result = append(result, protoFile)
		protoFileEntity := &protoFileEntity{
			imports: make([]string, 0),
		}
		// Parse the proto file using go-protoparser
		f, err := os.Open(protoFile)
		if err != nil {
			return err
		}
		defer f.Close()

		protoBytes, err := os.ReadFile(protoFile)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		buf := bytes.NewBuffer(protoBytes)
		parsedBody, err := protoparser.Parse(buf)
		if err != nil {
			return err
		}
		r.parsedProtoBody[protoFile] = parsedBody
		for _, body := range parsedBody.ProtoBody {
			switch b := body.(type) {
			case *protoparserparser.Import: // resolve relation for each imports
				importPath := b.Location
				importPath = strings.Trim(importPath, `"`)
				// TODO handle this better
				if strings.Contains(importPath, "google/protobuf") {
					continue
				}
				fullImportPath, err := r.findIfProtoExists(importPath)
				if err != nil {
					return err
				}
				protoFileEntity.imports = append(protoFileEntity.imports, fullImportPath)
				if err = dfs(fullImportPath); err != nil {
					return err
				}
			}
		}
		r.protoEntities[protoFile] = protoFileEntity
		return nil
	}
	// run dfs on the input proto path
	protoPath, err := r.findIfProtoExists(protoFile)
	if err != nil {
		return nil, err
	}
	if err := dfs(protoPath); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *Registry) findIfProtoExists(protoPath string) (string, error) {
	var (
		fullPath string
		fullProtoPath string
		err           error
	)
	protoPath = strings.Trim(protoPath, `"`)
	for _, dir := range r.ProtoDirectories {
		fullPath = path.Join(dir, protoPath)
		// Check if the path exists
		_, err = os.Stat(fullPath)
		if err == nil {
			fullProtoPath = fullPath
			break
		}
	}
	if fullProtoPath == ""{
		return "", fmt.Errorf("path does not exist: %s %w", fullPath, err)
	}
	if !strings.HasSuffix(fullProtoPath,".proto") {
		return "", fmt.Errorf("is not a .proto file %s %w", fullPath, err)
	}
	return fullProtoPath, nil
}

/*
This helper function will return the entity for any referenced type ,
Be it top/file,nested or imported entities.If not found will return an error
Ref - https://github.com/protocolbuffers/protobuf/blob/b7a5772caf08d62a20fd1bca258f501fa4db022c/src/google/protobuf/descriptor.proto#L186-L191
*/
func getReferencedType(typeName, prefix string, allResolvedEntities map[string]struct{}) (string, error) {
	// check if fully qualifed prefixed by dot
	if strings.HasPrefix(typeName, ".") {
		return getFullyQualifiedType(typeName, allResolvedEntities)
	}
	//  check if the entity is referenced to other packages via packageName
	if _, ok := allResolvedEntities[typeName]; ok {
		return typeName, nil
	}
	// try resolving from inner entities up till the parent package
	if result, ok := splitNameAndCheck(typeName, prefix, allResolvedEntities); ok {
		return result, nil
	}
	return "", fmt.Errorf("unable to resolve type name: %s", typeName)
}

// splitNameAndCheck splits the prefixName and tries to append the typeName and find the entity for resolution
// it also tries the find the entities defined using relative path
func splitNameAndCheck(typeName, prefix string, allResolvedEntities map[string]struct{}) (string, bool) {
	var (
		prefixSplit []string
		entityName  string
	)
	prefixSplit = strings.Split(prefix, ".")

	for len(prefixSplit) > 0 && prefixSplit[0] != "" {
		result := strings.Join(prefixSplit, ".")
		entityName = result + "." + typeName
		if _, ok := allResolvedEntities[entityName]; ok {
			return entityName, true
		}
		// Omit the last element in each iteration as we go level above to outer entity
		prefixSplit = prefixSplit[:len(prefixSplit)-1]
	}
	return "", false
}

func getFullyQualifiedType(typeName string, allResolvedEntities map[string]struct{}) (string, error) {

	typeName = strings.TrimPrefix(typeName, ".")
	if _, ok := allResolvedEntities[typeName]; ok {
		return typeName, nil
	}
	return "", fmt.Errorf("unbale to resolve full qualified prefixed with (.) type name: %s", typeName)
}
