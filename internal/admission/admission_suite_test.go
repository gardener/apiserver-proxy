// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0
package admission

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	v1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/admission/plugin/webhook/mutating"
	webhooktesting "k8s.io/apiserver/pkg/admission/plugin/webhook/testing"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var stopCtx, cancelFn = context.WithCancel(context.Background())

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Webhook Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var (
	stopCh = make(chan struct{})
	wh     *mutating.Plugin
)

var _ = BeforeSuite(func(done Done) {
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zap.Options{})))

	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	server := &webhook.Server{
		CertDir: filepath.Join("testdata"),
		Port:    10250,
	}

	Expect(Complete(server, "real-fqdn")).ToNot(HaveOccurred())

	go func() {
		err := server.Start(stopCtx)
		if err != nil {
			Expect(err).NotTo(HaveOccurred())
		}
	}()

	// wait for the webhook server to get ready
	dialer := &net.Dialer{Timeout: time.Second}
	addrPort := fmt.Sprintf("%s:%d", "localhost", 10250)
	Eventually(func() error {
		conn, err := tls.DialWithDialer(dialer, "tcp", addrPort, &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			return err
		}
		conn.Close()

		return nil
	}).Should(Succeed())

	webhookServer, err := url.Parse("https://localhost:10250")
	Expect(err).ToNot(HaveOccurred(), "failed to parse url")

	caCert, err := ioutil.ReadFile(filepath.Join("testdata", "ca.crt"))
	Expect(err).NotTo(HaveOccurred(), "ca.crt can be read")

	wh, err = mutating.NewMutatingWebhook(nil)
	Expect(err).ToNot(HaveOccurred(), "failed to create mutating webhook")

	client, informer := webhooktesting.NewFakeMutatingDataSource("foo", []v1.MutatingWebhook{{
		Name:                    "foo",
		NamespaceSelector:       &metav1.LabelSelector{},
		ObjectSelector:          &metav1.LabelSelector{},
		AdmissionReviewVersions: []string{"v1beta1"},
		Rules: []v1.RuleWithOperations{{
			Operations: []v1.OperationType{v1.OperationAll},
			Rule: v1.Rule{
				APIGroups:   []string{"*"},
				APIVersions: []string{"*"},
				Resources:   []string{"*/*"},
			},
		}},
		ClientConfig: v1.WebhookClientConfig{
			Service: &v1.ServiceReference{
				Name:      "webhook-test",
				Namespace: "default",
				Path:      pointer.StringPtr("/webhook/pod-apiserver-env"),
			},
			CABundle: caCert,
		},
	}}, nil)

	wh.SetAuthenticationInfoResolverWrapper(webhooktesting.Wrapper(webhooktesting.NewAuthenticationInfoResolver(new(int32))))
	wh.SetServiceResolver(webhooktesting.NewServiceResolver(*webhookServer))
	wh.SetExternalKubeClientSet(client)
	wh.SetExternalKubeInformerFactory(informer)

	informer.Start(stopCh)
	informer.WaitForCacheSync(stopCh)

	Expect(wh.ValidateInitialization()).ToNot(HaveOccurred(), "failed to validate initialization")

	close(done)
}, 60)

var _ = AfterSuite(func() {
	cancelFn()
	close(stopCh)
})

var _ = Describe("Pod admission", func() {
	var pod, expected *corev1.Pod

	BeforeEach(func() {
		pod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "test",
			}, Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name: "test",
					},
				},
			},
		}
		expected = pod.DeepCopy()
	})
	It("it adds extra environment variable", func() {
		expected.Spec.Containers[0].Env = []corev1.EnvVar{{Name: "KUBERNETES_SERVICE_HOST", Value: "real-fqdn"}}

		Expect(wh.Admit(context.TODO(), newPodAttribute(pod), webhooktesting.NewObjectInterfacesForTest())).NotTo(HaveOccurred())
		Expect(pod).To(Equal(expected))
	})

	It("it adds extra environment variable to init containers", func() {
		pod.Spec.InitContainers = []corev1.Container{{Name: "init"}}
		expected.Spec.Containers[0].Env = []corev1.EnvVar{{Name: "KUBERNETES_SERVICE_HOST", Value: "real-fqdn"}}
		expected.Spec.InitContainers = []corev1.Container{{
			Name: "init",
			Env: []corev1.EnvVar{{Name: "KUBERNETES_SERVICE_HOST", Value: "real-fqdn"}},
		}}

		Expect(wh.Admit(context.TODO(), newPodAttribute(pod), webhooktesting.NewObjectInterfacesForTest())).NotTo(HaveOccurred())
		Expect(pod).To(Equal(expected))
	})

	It("it doesn't override existing KUBERNETES_SERVICE_HOST variable", func() {
		pod.Spec.Containers[0].Env = []corev1.EnvVar{{Name: "KUBERNETES_SERVICE_HOST", Value: "baz"}}
		pod.Spec.Containers = append(pod.Spec.Containers, corev1.Container{Name: "test-2"})

		expected.Spec.Containers[0].Env = []corev1.EnvVar{{Name: "KUBERNETES_SERVICE_HOST", Value: "baz"}}
		expected.Spec.Containers = append(expected.Spec.Containers, corev1.Container{
			Name: "test-2", Env: []corev1.EnvVar{{Name: "KUBERNETES_SERVICE_HOST", Value: "real-fqdn"}},
		})

		Expect(wh.Admit(context.TODO(), newPodAttribute(pod), webhooktesting.NewObjectInterfacesForTest())).NotTo(HaveOccurred())
		Expect(pod).To(Equal(expected))
	})
})

func newPodAttribute(p *corev1.Pod) admission.Attributes {
	return admission.NewAttributesRecord(
		p,
		nil,
		corev1.SchemeGroupVersion.WithKind("Pod"),
		"test",
		"foo",
		corev1.SchemeGroupVersion.WithResource("pods"),
		"",
		admission.Create,
		&metav1.CreateOptions{},
		false,
		&user.DefaultInfo{
			Name:   "webhook-test",
			UID:    "webhook-test",
			Groups: nil,
			Extra:  nil,
		},
	)
}
