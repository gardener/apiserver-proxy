// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package admission

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	adm "sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// Complete adds the injector webhook to the server.
func Complete(server *webhook.Server, fqdn string) error {
	scheme := runtime.NewScheme()
	utilruntime.Must(corev1.AddToScheme(scheme))

	wh := adm.DefaultingWebhookFor(&PodMutator{fqdn: fqdn})

	if err := server.InjectFunc(func(i interface{}) error {
		return wh.InjectScheme(scheme)
	}); err != nil {
		return err
	}

	server.Register("/webhook/pod-apiserver-env", wh)

	return nil
}

var _ webhook.Defaulter = &PodMutator{}

// PodMutator wraps Pod to be used as webhook.Defaulter.
type PodMutator struct {
	*corev1.Pod
	fqdn string
}

func (p *PodMutator) GetObjectKind() schema.ObjectKind {
	return &runtime.TypeMeta{APIVersion: "v1", Kind: "Pod"}
}

func (p *PodMutator) DeepCopyObject() runtime.Object {
	cpPod := &corev1.Pod{}
	if p.Pod != nil {
		cpPod = p.Pod.DeepCopy()
	}

	return &PodMutator{
		Pod:  cpPod,
		fqdn: p.fqdn,
	}
}

// Default implements webhook.Defaulter so a webhook will be registered for the type.
func (p *PodMutator) Default() {
	if p.Pod == nil {
		return
	}

	for i, c := range p.Pod.Spec.Containers {
		if len(c.Env) == 0 || !hasEnv(c.Env) {
			if c.Env == nil {
				c.Env = make([]corev1.EnvVar, 0, 1)
			}

			p.Pod.Spec.Containers[i].Env = append(p.Pod.Spec.Containers[i].Env, corev1.EnvVar{
				Name:      "KUBERNETES_SERVICE_HOST",
				Value:     p.fqdn,
				ValueFrom: nil,
			})
		}
	}
}

func hasEnv(envs []corev1.EnvVar) bool {
	found := false

	for _, e := range envs {
		if e.Name == "KUBERNETES_SERVICE_HOST" {
			return true
		}
	}

	return found
}
