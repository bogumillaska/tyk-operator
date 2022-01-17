package integration

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/TykTechnologies/tyk-operator/pkg/environmet"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

var (
	e       environmet.Env
	testenv env.Environment
)

const reconcileDelay = time.Second * 5
const ctxNSKey = "test-ns"

func TestMain(t *testing.M) {
	e.Parse()

	if e.Mode == "" {
		log.Fatal("Missing TYK_MODE")
	}

	testenv = env.New()

	testenv.Setup(
		setupk8s,
		setupTyk,
		setupE2E,
		setupMultiTenancy,
	).Finish(
		teardownMultiTenancy,
		teardownE2E,
		teardownTyk,
		teardownk8s,
	).BeforeTest(
		createNamespace,
	).AfterTest(
		deleteNamespace,
	)

	os.Exit(testenv.Run(t))
}

func createNamespace(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
	name := envconf.RandomName("tyk-operator", 16)

	ctx = context.WithValue(ctx, ctxNSKey, name)

	nsObj := v1.Namespace{}
	nsObj.Name = name
	return ctx, cfg.Client().Resources().Create(ctx, &nsObj)
}

func deleteNamespace(ctx context.Context, envconf *envconf.Config) (context.Context, error) {
	name := ctx.Value(ctxNSKey)

	nsObj := v1.Namespace{}
	nsObj.Name = name.(string)
	return ctx, envconf.Client().Resources().Delete(ctx, &nsObj)
}