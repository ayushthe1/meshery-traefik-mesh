package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/layer5io/meshery-adapter-library/adapter"
	"github.com/layer5io/meshkit/utils"
	"github.com/layer5io/meshkit/utils/manifests"
	walker "github.com/layer5io/meshkit/utils/walker"
	smp "github.com/layer5io/service-mesh-performance/spec"
)

var DefaultVersion string
var DefaultURL string
var DefaultGenerationMethod string
var WorkloadPath string
var MeshModelPath string
var AllVersions []string
var CRDNames []string

var meshmodelmetadata = map[string]interface{}{
	"Primary Color":   "#9D0FB0",
	"Secondary Color": "#e281f0",
	"Shape":           "circle",
	"Logo URL":        "",
	"SVG_Color":       "",
	"SVG_White":       "<svg width=\"32\" height=\"32\" viewBox=\"0 0 32 32\" fill=\"none\" xmlns=\"http://www.w3.org/2000/svg\"><g clip-path=\"url(#a)\"><path d=\"m11.54 6.143.046.025 3.928 2.226a.878.878 0 0 0 .865 0l3.896-2.202a1.315 1.315 0 0 1 1.792.482 1.336 1.336 0 0 1-.45 1.808l-.045.027-2.249 1.273a.442.442 0 0 0-.224.386.446.446 0 0 0 .224.386l6.072 3.442a.875.875 0 0 0 .865 0l3.756-2.129a1.315 1.315 0 0 1 1.796.478 1.335 1.335 0 0 1-.453 1.812l-.045.027-2.115 1.196a.442.442 0 0 0-.164.61c.04.067.096.123.164.162l2.125 1.204a1.329 1.329 0 0 1 .518 1.79 1.316 1.316 0 0 1-1.771.551l-.046-.025-3.766-2.135a.877.877 0 0 0-.865 0l-6.14 3.48a.441.441 0 0 0-.164.61.44.44 0 0 0 .164.161l2.537 1.438a1.336 1.336 0 0 1 .51 1.785 1.318 1.318 0 0 1-1.763.556l-.046-.025-4.177-2.369a.878.878 0 0 0-.865 0l-4.215 2.392a1.315 1.315 0 0 1-1.79-.48 1.335 1.335 0 0 1 .447-1.81l.045-.026 2.572-1.46a.44.44 0 0 0 .167-.604.432.432 0 0 0-.167-.168l-6.084-3.448a.878.878 0 0 0-.866 0l-3.582 2.027a1.316 1.316 0 0 1-1.794-.48 1.336 1.336 0 0 1 .453-1.81l.045-.027 1.937-1.096a.44.44 0 0 0 .225-.386.444.444 0 0 0-.225-.386L.684 14.314a1.336 1.336 0 0 1-.51-1.785 1.315 1.315 0 0 1 1.762-.556l.047.025 3.579 2.029a.876.876 0 0 0 .864 0l6.143-3.476a.442.442 0 0 0 .225-.386.444.444 0 0 0-.225-.386l-2.281-1.295a1.336 1.336 0 0 1-.51-1.785c.164-.305.44-.534.769-.638.329-.104.685-.074.993.083v-.001Zm3.973 5.793-6.144 3.476a.442.442 0 0 0-.165.61c.04.068.096.124.165.163l6.08 3.446a.876.876 0 0 0 .866 0l6.138-3.48a.442.442 0 0 0 .164-.609.443.443 0 0 0-.164-.162l-6.076-3.444a.877.877 0 0 0-.865 0Z\" fill=\"#fff\"/></g><defs><clipPath id=\"a\"><path fill=\"#fff\" d=\"M0 0h32v32H0z\"/></clipPath></defs></svg>",
}

var MeshModelConfig = adapter.MeshModelConfig{ //Move to build/config.go
	Category:    "Orchestration & Management",
	SubCategory: "Service Mesh",
	Metadata:    meshmodelmetadata,
}

// NewConfig creates the configuration for creating components
func NewConfig(version string) manifests.Config {
	return manifests.Config{
		Name:        smp.ServiceMesh_Type_name[int32(smp.ServiceMesh_TRAEFIK_MESH)],
		MeshVersion: version,
		CrdFilter: manifests.NewCueCrdFilter(manifests.ExtractorPaths{
			NamePath:    "spec.names.kind",
			IdPath:      "spec.names.kind",
			VersionPath: "spec.versions[0].name",
			GroupPath:   "spec.group",
			SpecPath:    "spec.versions[0].schema.openAPIV3Schema.properties.spec"}, false),
		ExtractCrds: func(manifest string) []string {
			crds := strings.Split(manifest, "---")
			return crds
		},
	}
}

func init() {
	wd, _ := os.Getwd()
	WorkloadPath = filepath.Join(wd, "templates", "oam", "workloads")
	MeshModelPath = filepath.Join(wd, "templates", "meshmodel", "components")
	AllVersions, _ = utils.GetLatestReleaseTagsSorted("traefik", "mesh")
	if len(AllVersions) == 0 {
		return
	}
	DefaultVersion = AllVersions[len(AllVersions)-1]
	DefaultGenerationMethod = adapter.Manifests

	//Get all the crd names
	w := walker.NewGithub()
	err := w.Owner("traefik").
		Repo("mesh-helm-chart").
		Branch("master").
		Root("mesh/crds/**").
		RegisterFileInterceptor(func(gca walker.GithubContentAPI) error {
			if gca.Content != "" {
				CRDNames = append(CRDNames, gca.Name)
			}
			return nil
		}).Walk()
	if err != nil {
		fmt.Println("Could not find CRD names. Will fail component creation...", err.Error())
	}
	DefaultURL = "https://raw.githubusercontent.com/traefik/mesh-helm-chart/" + "master" + "/mesh/crds/"
}
