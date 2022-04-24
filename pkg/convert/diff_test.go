package convert

import (
	"strings"
	"testing"
)

const input = `---
# Source: cert-manager/templates/cainjector-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cert-manager-cainjector
  namespace: "cert-manager"
  labels:
    app: cainjector
    app.kubernetes.io/name: cainjector
    app.kubernetes.io/instance: cert-manager
    app.kubernetes.io/component: "cainjector"
    app.kubernetes.io/version: "v1.8.0"
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: cainjector
      app.kubernetes.io/instance: cert-manager
      app.kubernetes.io/component: "cainjector"
  template:
    metadata:
      labels:
        app: cainjector
        app.kubernetes.io/name: cainjector
        app.kubernetes.io/instance: cert-manager
        app.kubernetes.io/component: "cainjector"
        app.kubernetes.io/version: "v1.8.0"
    spec:
      serviceAccountName: cert-manager-cainjector
      securityContext:
        runAsNonRoot: true
      containers:
        - name: cert-manager
          image: "quay.io/jetstack/cert-manager-cainjector:v1.8.0"
          imagePullPolicy: IfNotPresent
          args:
            - --v=2
            - --leader-election-namespace=kube-system
          env:
            - name: POD_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
          securityContext:
            allowPrivilegeEscalation: false
      nodeSelector:
        kubernetes.io/os: linux
---
# Source: cert-manager/templates/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cert-manager
  namespace: "cert-manager"
  labels:
    app: cert-manager
    app.kubernetes.io/name: cert-manager
    app.kubernetes.io/instance: cert-manager
    app.kubernetes.io/component: "controller"
    app.kubernetes.io/version: "v1.8.0"
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: cert-manager
      app.kubernetes.io/instance: cert-manager
      app.kubernetes.io/component: "controller"
  template:
    metadata:
      labels:
        app: cert-manager
        app.kubernetes.io/name: cert-manager
        app.kubernetes.io/instance: cert-manager
        app.kubernetes.io/component: "controller"
        app.kubernetes.io/version: "v1.8.0"
      annotations:
        prometheus.io/path: "/metrics"
        prometheus.io/scrape: 'true'
        prometheus.io/port: '9402'
    spec:
      serviceAccountName: cert-manager
      securityContext:
        
        runAsNonRoot: true
      containers:
        - name: cert-manager
          image: "quay.io/jetstack/cert-manager-controller:v1.8.0"
          imagePullPolicy: IfNotPresent
          args:
            - --v=2
            - --cluster-resource-namespace=$(POD_NAMESPACE)
            - --leader-election-namespace=kube-system
          ports:
            - containerPort: 9402
              name: http-metrics
              protocol: TCP
          securityContext:
            allowPrivilegeEscalation: false
          env:
            - name: POD_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
      nodeSelector:
        kubernetes.io/os: linux
---
# Source: cert-manager/templates/webhook-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cert-manager-webhook
  namespace: "cert-manager"
  labels:
    app: webhook
    app.kubernetes.io/name: webhook
    app.kubernetes.io/instance: cert-manager
    app.kubernetes.io/component: "webhook"
    app.kubernetes.io/version: "v1.8.0"
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: webhook
      app.kubernetes.io/instance: cert-manager
      app.kubernetes.io/component: "webhook"
  template:
    metadata:
      labels:
        app: webhook
        app.kubernetes.io/name: webhook
        app.kubernetes.io/instance: cert-manager
        app.kubernetes.io/component: "webhook"
        app.kubernetes.io/version: "v1.8.0"
    spec:
      serviceAccountName: cert-manager-webhook
      securityContext:
        runAsNonRoot: true
      containers:
        - name: cert-manager
          image: "quay.io/jetstack/cert-manager-webhook:v1.8.0"
          imagePullPolicy: IfNotPresent
          args:
            - --v=2
            - --secure-port=10250
            - --dynamic-serving-ca-secret-namespace=$(POD_NAMESPACE)
            - --dynamic-serving-ca-secret-name=cert-manager-webhook-ca
            - --dynamic-serving-dns-names=cert-manager-webhook,cert-manager-webhook.cert-manager,cert-manager-webhook.cert-manager.svc
          ports:
            - name: https
              protocol: TCP
              containerPort: 10250
          livenessProbe:
            httpGet:
              path: /livez
              port: 6080
              scheme: HTTP
            initialDelaySeconds: 60
            periodSeconds: 10
            timeoutSeconds: 1
            successThreshold: 1
            failureThreshold: 3
          readinessProbe:
            httpGet:
              path: /healthz
              port: 6080
              scheme: HTTP
            initialDelaySeconds: 5
            periodSeconds: 5
            timeoutSeconds: 1
            successThreshold: 1
            failureThreshold: 3
          securityContext:
            allowPrivilegeEscalation: false
          env:
            - name: POD_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
      nodeSelector:
        kubernetes.io/os: linux`

func Test_Diff(t *testing.T) {
	objects, err := ParseYAMLFileIntoJSONObjects([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	pg, err := NewPatchGenerator("/Users/amannmalik/IdeaProjects/kubernetes/api/openapi-spec/v3")
	if err != nil {
		t.Fatal(err)
	}
	results, err := pg.Execute(objects)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) < 1 {
		t.Fatal(err)
	}
	result := results[0]

	baseYAML, err := result.GetBaseYAML()
	if err != nil {
		t.Fatal(err)
	}
	if len(baseYAML) < 3 {
		t.Fatal(err)
	}
	patches, err := result.GetPatchYAMLs()
	if err != nil {
		t.Fatal(err)
	}
	if len(patches) != 3 {
		t.Fatal(err)
	}

	t.Log(string(baseYAML))
	consolidated := []string{}
	for _, patch := range patches {
		consolidated = append(consolidated, string(patch))
	}
	t.Log(strings.Join(consolidated, "---\n"))
}
