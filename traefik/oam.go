package traefik

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/layer5io/meshery-adapter-library/meshes"
	"github.com/layer5io/meshery-traefik-mesh/internal/config"
	"github.com/layer5io/meshkit/models/oam/core/v1alpha1"
	"gopkg.in/yaml.v2"
)

// CompHandler is the type for functions which can handle OAM components
type CompHandler func(*Mesh, v1alpha1.Component, bool, []string) (string, error)

// HandleComponents handles the processing of OAM components
func (mesh *Mesh) HandleComponents(comps []v1alpha1.Component, isDel bool, kubeconfigs []string) (string, error) {
	var errs []error
	var msgs []string
	stat1 := "deploying"
	stat2 := "deployed"
	if isDel {
		stat1 = "removing"
		stat2 = "removed"
	}
	compFuncMap := map[string]CompHandler{
		"TraefikMesh": handleComponentTraefikMesh,
	}

	for _, comp := range comps {
		ee := &meshes.EventsResponse{
			OperationId:   uuid.New().String(),
			Component:     config.ServerConfig["type"],
			ComponentName: config.ServerConfig["name"],
		}
		fnc, ok := compFuncMap[comp.Spec.Type]
		if !ok {
			msg, err := handleTraefikCoreComponent(mesh, comp, isDel, "", "", kubeconfigs)
			if err != nil {
				ee.Summary = fmt.Sprintf("Error while %s %s", stat1, comp.Spec.Type)
				mesh.streamErr(ee.Summary, ee, err)
				errs = append(errs, err)
				continue
			}
			ee.Summary = fmt.Sprintf("%s %s successfully", comp.Spec.Type, stat2)
			ee.Details = fmt.Sprintf("The %s is now %s.", comp.Spec.Type, stat2)
			mesh.StreamInfo(ee)
			msgs = append(msgs, msg)
			continue
		}

		msg, err := fnc(mesh, comp, isDel, kubeconfigs)
		if err != nil {
			ee.Summary = fmt.Sprintf("Error while %s %s", stat1, comp.Spec.Type)
			mesh.streamErr(ee.Summary, ee, err)
			errs = append(errs, err)
			continue
		}
		ee.Summary = fmt.Sprintf("%s %s %s successfully", comp.Name, comp.Spec.Type, stat2)
		ee.Details = fmt.Sprintf("The %s %s is now %s.", comp.Name, comp.Spec.Type, stat2)
		mesh.StreamInfo(ee)
		msgs = append(msgs, msg)
	}

	if err := mergeErrors(errs); err != nil {
		return mergeMsgs(msgs), err
	}

	return mergeMsgs(msgs), nil
}

// HandleApplicationConfiguration handles the processing of OAM application configuration
func (mesh *Mesh) HandleApplicationConfiguration(config v1alpha1.Configuration, isDel bool, kubeconfigs []string) (string, error) {
	var errs []error
	var msgs []string
	for _, comp := range config.Spec.Components {
		for _, trait := range comp.Traits {
			msgs = append(msgs, fmt.Sprintf("applied trait \"%s\" on service \"%s\"", trait.Name, comp.ComponentName))
		}
	}

	if err := mergeErrors(errs); err != nil {
		return mergeMsgs(msgs), err
	}

	return mergeMsgs(msgs), nil
}

func handleComponentTraefikMesh(mesh *Mesh, comp v1alpha1.Component, isDel bool, kubeconfigs []string) (string, error) {
	version := comp.Spec.Version
	msg, err := mesh.installTraefikMesh(isDel, version, comp.Namespace, kubeconfigs)
	if err != nil {
		return fmt.Sprintf("%s: %s", comp.Name, msg), err
	}

	return fmt.Sprintf("%s: %s", comp.Name, msg), nil
}

func handleTraefikCoreComponent(
	mesh *Mesh,
	comp v1alpha1.Component,
	isDel bool,
	apiVersion,
	kind string,
	kubeconfigs []string) (string, error) {
	if apiVersion == "" {
		apiVersion = getAPIVersionFromComponent(comp)
		if apiVersion == "" {
			return "", ErrTraefikCoreComponentFail(fmt.Errorf("failed to get API Version for: %s", comp.Name))
		}
	}

	if kind == "" {
		kind = getKindFromComponent(comp)
		if kind == "" {
			return "", ErrTraefikCoreComponentFail(fmt.Errorf("failed to get kind for: %s", comp.Name))
		}
	}

	component := map[string]interface{}{
		"apiVersion": apiVersion,
		"kind":       kind,
		"metadata": map[string]interface{}{
			"name":        comp.Name,
			"annotations": comp.Annotations,
			"labels":      comp.Labels,
		},
		"spec": comp.Spec.Settings,
	}

	// Convert to yaml
	yamlByt, err := yaml.Marshal(component)
	if err != nil {
		err = ErrParseTraefikCoreComponent(err)
		mesh.Log.Error(err)
		return "", err
	}

	msg := fmt.Sprintf("created %s \"%s\" in namespace \"%s\"", kind, comp.Name, comp.Namespace)
	if isDel {
		msg = fmt.Sprintf("deleted %s config \"%s\" in namespace \"%s\"", kind, comp.Name, comp.Namespace)
	}

	return msg, mesh.applyManifest(yamlByt, isDel, comp.Namespace, kubeconfigs)
}

func getAPIVersionFromComponent(comp v1alpha1.Component) string {
	return comp.Annotations["pattern.meshery.io.mesh.workload.k8sAPIVersion"]
}

func getKindFromComponent(comp v1alpha1.Component) string {
	return comp.Annotations["pattern.meshery.io.mesh.workload.k8sKind"]
}

func mergeErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}

	var errMsgs []string

	for _, err := range errs {
		errMsgs = append(errMsgs, err.Error())
	}

	return fmt.Errorf(strings.Join(errMsgs, "\n"))
}

func mergeMsgs(strs []string) string {
	return strings.Join(strs, "\n")
}
