package cert

import (
	"context"
	"errors"
	"fmt"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	authConfigNamespace = metav1.NamespaceSystem

	// authenticationConfigMapName is the name of ConfigMap in the kube-system namespace holding the root certificate
	// bundle to use to verify client certificates on incoming requests before trusting usernames in headers specified
	// by --requestheader-username-headers. This is created in the cluster by the kube-apiserver.
	// "WARNING: generally do not depend on authorization being already done for incoming requests.")
	authConfigName = "extension-apiserver-authentication"

	authRoleName = "extension-apiserver-authentication-reader"

	clientCAFileKey = "client-ca-file"
)

func GetClientCA(client kubernetes.Interface) (clientCA []byte, err error) {
	if client == nil {
		err = errors.New("no client specified")
	}

	configmap, getErr := client.CoreV1().ConfigMaps(authConfigNamespace).Get(context.TODO(), authConfigName, metav1.GetOptions{})
	if getErr != nil {
		if k8serrors.IsForbidden(getErr) {
			err = fmt.Errorf("Unable to get configmap/%s in %s.  Usually fixed by 'kubectl create rolebinding -n %s ROLEBINDING_NAME --role=%s --serviceaccount=YOUR_NS:YOUR_SA'",
				authConfigName, authConfigNamespace, authConfigNamespace, authRoleName)
			return
		}

		err = fmt.Errorf("failed to get client CA bundle - %s", getErr.Error())
		return
	}

	data, ok := configmap.Data[clientCAFileKey]
	if !ok || len(data) == 0 {
		err = fmt.Errorf("did not find client CA in configmap/%s", authConfigName)
		return
	}

	clientCA = []byte(data)
	return
}
