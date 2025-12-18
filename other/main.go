package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	// Replace with your Service Account details
	serviceAccountName = "test"
	namespace          = "default"
	tokenAudience      = "my-external-service" // Define the audience for your token
	tokenExpiration    = int64(3600)           // 1 hour
)

// OIDCDiscovery represents the OIDC Discovery Document structure
type OIDCDiscovery struct {
	Issuer  string `json:"issuer"`
	JWKSURI string `json:"jwks_uri"`
	// ... other OIDC fields
}

func main() {
	// 1. Load Kubernetes configuration
	// Uses the KUBECONFIG environment variable or default location
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		kubeconfigPath = clientcmd.RecommendedHomeFile
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		panic(fmt.Errorf("error building kubeconfig: %w", err))
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(fmt.Errorf("error creating clientset: %w", err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 2. Request the Service Account Token
	fmt.Printf("Requesting token for Service Account %s/%s...\n", namespace, serviceAccountName)

	tokenReq := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			Audiences:         []string{tokenAudience},
			ExpirationSeconds: &tokenExpiration,
		},
	}

	tokenResult, err := clientset.CoreV1().ServiceAccounts(namespace).CreateToken(ctx, serviceAccountName, tokenReq, metav1.CreateOptions{})
	if err != nil {
		panic(fmt.Errorf("error creating token: %w", err))
	}

	serviceAccountToken := tokenResult.Status.Token
	fmt.Println("Successfully retrieved Service Account Token: %s", serviceAccountToken)
	// fmt.Printf("Token (JWT): %s\n", serviceAccountToken) // Uncomment to see the token itself

	// 3. Determine the OIDC Issuer URL
	// Kubernetes Service Account JWTs have a standard issuer claim:
	// The issuer is typically configured on the API server with the
	// --service-account-issuer flag.
	// You may need to manually configure the expected issuer URL based on your cluster setup.
	// For most clusters, this is the cluster's API server address.
	issuerURL := config.Host

	fmt.Printf("Deduced OIDC Issuer URL: %s\n", issuerURL)
	oidcDiscoveryURL := fmt.Sprintf("%s/.well-known/openid-configuration", issuerURL)

	// 4. Fetch the OIDC Discovery Document and JWKS
	fmt.Printf("Fetching OIDC Discovery Document from %s...\n", oidcDiscoveryURL)

	// To perform the request, we'll need an HTTP client, which for a self-signed
	// Kubernetes cluster certificate, should use the cluster's CA bundle.
	httpClient, err := rest.HTTPClientFor(config)
	if err != nil {
		panic(fmt.Errorf("error creating HTTP client: %w", err))
	}

	// RESTClient().Get().AbsPath("/.well-known/openid-configuration")

	resp, err := httpClient.Get(oidcDiscoveryURL)
	if err != nil {
		panic(fmt.Errorf("error fetching OIDC Discovery Document: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		panic(fmt.Errorf("failed to fetch OIDC Discovery Document, status code: %d, body: %s", resp.StatusCode, string(bodyBytes)))
	}

	var discoveryDoc OIDCDiscovery
	if err := json.NewDecoder(resp.Body).Decode(&discoveryDoc); err != nil {
		panic(fmt.Errorf("error decoding OIDC Discovery Document: %w", err))
	}

	fmt.Printf("OIDC Discovery Document Issuer: %s\n", discoveryDoc.Issuer)
	fmt.Printf("JWKS URI (for public keys): %s\n", discoveryDoc.JWKSURI)

	discoveryDoc.JWKSURI = strings.Replace(discoveryDoc.JWKSURI, "192.168.126.130", "127.0.0.1", 1)
	// ---
	// You can now use discoveryDoc.JWKSURI to fetch the public keys (JWKS).
	// Fetching the JWKS is similar to fetching the discovery document.
	// ---
	fmt.Printf("\nFetching JWKS from %s...\n", discoveryDoc.JWKSURI)

	jwksResp, err := httpClient.Get(discoveryDoc.JWKSURI)
	if err != nil {
		panic(fmt.Errorf("error fetching JWKS: %w", err))
	}
	defer jwksResp.Body.Close()

	if jwksResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(jwksResp.Body)
		panic(fmt.Errorf("failed to fetch JWKS, status code: %d, body: %s", jwksResp.StatusCode, string(bodyBytes)))
	}

	jwksBody, err := io.ReadAll(jwksResp.Body)
	if err != nil {
		panic(fmt.Errorf("error reading JWKS body: %w", err))
	}

	var data map[string]interface{}

	err = json.Unmarshal(jwksBody, &data)
	if err != nil {
		panic(fmt.Errorf("error parsing response: %s\n", err))
	}
	// The jwksBody contains the JSON Web Key Set (JWKS) with the public keys
	// required to validate the signature of the Service Account JWT.
	fmt.Println("Successfully retrieved JWKS (public keys).")
	// fmt.Println(string(jwksBody)) // Uncomment to see the JWKS

	config2 := rest.CopyConfig(config)
	config2.BearerToken = tokenResult.Status.Token
	config2.TLSClientConfig.KeyData = nil
	config2.TLSClientConfig.CertData = nil

	clientset2, err := kubernetes.NewForConfig(config2)
	if err != nil {
		panic(fmt.Errorf("error creating clientset: %w", err))
	}

	_, err = clientset2.CoreV1().ServiceAccounts("default").List(ctx, metav1.ListOptions{})
	if err != nil {
		panic(fmt.Errorf("error list: %w", err))
	}
}
