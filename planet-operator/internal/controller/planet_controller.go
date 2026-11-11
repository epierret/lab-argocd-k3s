package controller

import (
	"context"
	"fmt"
	"html"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	spacev1alpha1 "github.com/epierret/planet-operator/api/v1alpha1"
)

const (
	// defaultImage sert la page générée quand la planète ne précise pas d'image.
	defaultImage = "nginx:1.27-alpine"
	// defaultPort est le port de nginx ; les images personnalisées écoutent sur 8080.
	defaultPort  = 80
	customPort   = 8080
	pageKey      = "index.html"
	pageMountDir = "/usr/share/nginx/html"
)

// PlanetReconciler réconcilie les ressources Planet.
type PlanetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=space.lab.example.com,resources=planets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=space.lab.example.com,resources=planets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=space.lab.example.com,resources=planets/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services;configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile crée ou met à jour le ConfigMap (si besoin), le Deployment et le Service,
// puis reflète l'état dans le status de la Planet.
func (r *PlanetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var planet spacev1alpha1.Planet
	if err := r.Get(ctx, req.NamespacedName, &planet); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	labels := map[string]string{
		"app.kubernetes.io/name":       "planet",
		"space.lab.example.com/planet": planet.Name,
	}

	image, port := defaultImage, int32(defaultPort)
	if planet.Spec.Image != "" {
		image, port = planet.Spec.Image, customPort
	} else {
		if err := r.ensurePage(ctx, &planet, labels); err != nil {
			return ctrl.Result{}, err
		}
	}

	if err := r.ensureDeployment(ctx, &planet, image, port, labels); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.ensureService(ctx, &planet, port, labels); err != nil {
		return ctrl.Result{}, err
	}

	planet.Status.Phase = "Ready"
	if err := r.Status().Update(ctx, &planet); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("planète réconciliée", "planet", planet.Name, "image", image)
	return ctrl.Result{}, nil
}

// ensurePage crée le ConfigMap qui contient la page HTML générée à partir de la spec.
func (r *PlanetReconciler) ensurePage(ctx context.Context, planet *spacev1alpha1.Planet, labels map[string]string) error {
	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: planet.Name, Namespace: planet.Namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
		cm.Labels = labels
		cm.Data = map[string]string{pageKey: renderPage(planet)}
		return controllerutil.SetControllerReference(planet, cm, r.Scheme)
	})
	return err
}

// ensureDeployment crée ou met à jour le Deployment de la planète.
func (r *PlanetReconciler) ensureDeployment(ctx context.Context, planet *spacev1alpha1.Planet, image string, port int32, labels map[string]string) error {
	dep := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: planet.Name, Namespace: planet.Namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, dep, func() error {
		dep.Labels = labels
		replicas := int32(1)
		dep.Spec = appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "planet",
						Image: image,
						Ports: []corev1.ContainerPort{{ContainerPort: port}},
						SecurityContext: &corev1.SecurityContext{
							AllowPrivilegeEscalation: boolPtr(false),
						},
					}},
				},
			},
		}
		if planet.Spec.Image == "" {
			mountPage(&dep.Spec.Template.Spec, planet.Name)
		}
		return controllerutil.SetControllerReference(planet, dep, r.Scheme)
	})
	return err
}

// ensureService expose la planète dans le cluster sur le port 80.
func (r *PlanetReconciler) ensureService(ctx context.Context, planet *spacev1alpha1.Planet, port int32, labels map[string]string) error {
	svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: planet.Name, Namespace: planet.Namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
		svc.Labels = labels
		svc.Spec = corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{{
				Name:       "http",
				Port:       80,
				TargetPort: intstr.FromInt32(port),
			}},
		}
		return controllerutil.SetControllerReference(planet, svc, r.Scheme)
	})
	return err
}

// SetupWithManager branche le contrôleur sur les Planet et sur les ressources qu'il possède.
func (r *PlanetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&spacev1alpha1.Planet{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}

// renderPage génère la page HTML. Toutes les valeurs saisies sont échappées.
func renderPage(planet *spacev1alpha1.Planet) string {
	return fmt.Sprintf(`<!doctype html>
<html lang="fr">
<head><meta charset="utf-8"><title>%[1]s</title></head>
<body>
<h1>%[1]s</h1>
<ul>
<li>Masse : %[2]g kg</li>
<li>Rayon : %[3]g km</li>
<li>Orbite : %[4]d jours</li>
</ul>
<p>%[5]s</p>
</body>
</html>
`,
		html.EscapeString(planet.Spec.Name),
		planet.Spec.MassKg,
		planet.Spec.RadiusKm,
		planet.Spec.OrbitDays,
		html.EscapeString(planet.Spec.Message),
	)
}

// mountPage monte le ConfigMap de la page dans le conteneur nginx.
func mountPage(spec *corev1.PodSpec, name string) {
	spec.Volumes = []corev1.Volume{{
		Name: "page",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: name},
			},
		},
	}}
	spec.Containers[0].VolumeMounts = []corev1.VolumeMount{{
		Name:      "page",
		MountPath: pageMountDir,
		ReadOnly:  true,
	}}
}

func boolPtr(b bool) *bool {
	return &b
}
