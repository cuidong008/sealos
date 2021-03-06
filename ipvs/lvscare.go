package ipvs

import (
	"github.com/pkg/errors"
	"github.com/wonderivan/logger"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)
const defaultImage = "fanux/lvscare:latest"

// return lvscare static pod yaml
func LvsStaticPodYaml(vip string, masters []string, image string) string {
	if image == "" {
		image = defaultImage
	}
	if vip == "" || len(masters) == 0 {
		return ""
	}
	args := []string{"care","--vs",vip+":6443","--health-path", "/healthz","--health-schem","https"}
	for _,m := range masters {
		args = append(args,"--rs")
		args = append(args, m + ":6443")
	}
	flag := true
	pod := ComponentPod(v1.Container{
		Name:                     "kube-sealyun-lvscare",
		Image:                    image,
		Command:                  []string{"/usr/bin/lvscare"},
		Args:                     args,
		ImagePullPolicy:          v1.PullIfNotPresent,
		SecurityContext:          &v1.SecurityContext{Privileged:&flag},
	})
	yaml,err := PodToYaml(pod)
	if err != nil {
		logger.Error("decode lvscare static pod yaml failed %s",err)
		return ""
	}
	return string(yaml)
}

func PodToYaml(pod v1.Pod)([]byte,error) {
	codecs := scheme.Codecs
	gv := v1.SchemeGroupVersion
	const mediaType = runtime.ContentTypeYAML
	info, ok := runtime.SerializerInfoForMediaType(codecs.SupportedMediaTypes(), mediaType)
	if !ok {
		return []byte{}, errors.Errorf("unsupported media type %q", mediaType)
	}

	encoder := codecs.EncoderForVersion(info.Serializer, gv)
	return runtime.Encode(encoder, &pod)
}

// ComponentPod returns a Pod object from the container and volume specifications
func ComponentPod(container v1.Container) v1.Pod {
	return v1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      container.Name,
			Namespace: metav1.NamespaceSystem,
			// The component and tier labels are useful for quickly identifying the control plane Pods when doing a .List()
			// against Pods in the kube-system namespace. Can for example be used together with the WaitForPodsWithLabel function
			Labels: map[string]string{"component": container.Name, "tier": "control-plane"},
		},
		Spec: v1.PodSpec{
			Containers:        []v1.Container{container},
			PriorityClassName: "system-cluster-critical",
			HostNetwork:       true,
		},
	}
}
