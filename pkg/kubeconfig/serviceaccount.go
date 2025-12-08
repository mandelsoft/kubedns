package kubeconfig

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// extractTokenFromTokenRequest uses the recommended method for modern Kubernetes (v1.24+).
func extractTokenFromTokenRequest(clientset *kubernetes.Clientset, saName, saNamespace string) (string, string, error) {
	ctx := context.Background()

	// 1. Create a TokenRequest object with a desired expiration
	expiration := int64(time.Hour * 24)
	tokenRequest, err := clientset.CoreV1().ServiceAccounts(saNamespace).
		CreateToken(ctx, saName, &v1.TokenRequest{
			Spec: v1.TokenRequestSpec{
				ExpirationSeconds: &expiration,
			},
		}, metav1.CreateOptions{})

	if err != nil {
		return "", "", fmt.Errorf("failed to create TokenRequest: %w", err)
	}

	// 2. TokenRequest does NOT provide the CA certificate.
	// We must retrieve the Service Account to find the Secret containing the CA.
	sa, err := clientset.CoreV1().ServiceAccounts(saNamespace).Get(ctx, saName, metav1.GetOptions{})
	if err != nil {
		return "", "", fmt.Errorf("failed to get ServiceAccount to find CA secret: %w", err)
	}

	if len(sa.Secrets) == 0 {
		return "", "", fmt.Errorf("service account %s has no associated secrets (required for legacy token extraction)", saName)
	}

	// 3. Use the first secret listed
	secretName := sa.Secrets[0].Name

	// 4. Get the Secret
	secret, err := clientset.CoreV1().Secrets(saNamespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return "", "", fmt.Errorf("failed to get token secret %s: %w", secretName, err)
	}

	// 4. Extract token and CA
	/* legacy
	tokenData, ok := secret.Data["token"]
	if !ok {
		return "", "", fmt.Errorf("secret %s does not contain 'token' key", secretName)
	}
	 */

	caData, ok := secret.Data["ca.crt"]
	if !ok {
		return "", "", fmt.Errorf("secret %s does not contain 'ca.crt' key", secretName)
	}
	return tokenRequest.Status.Token, string(caData), nil
}
