package integration

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/TykTechnologies/tyk-operator/api/model"
	"github.com/TykTechnologies/tyk-operator/api/v1alpha1"
	"github.com/matryer/is"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const retryOperationTimeout = 10 * time.Minute

func TestOperatorContextCreate(t *testing.T) {
	opCreate := features.New("Operator Context").
		Assess("Create", func(ctx context.Context, t *testing.T, envConf *envconf.Config) context.Context {
			testNS := ctx.Value(ctxNSKey).(string)
			is := is.New(t)

			// create api definition
			_, err := createTestAPIDef(ctx, testNS, envConf)
			is.NoErr(err) // failed to create apiDefinition

			_, err = createTestOperatorContext(ctx, testNS, envConf)
			is.NoErr(err) // failed to create operatorcontext

			err = retryOperation(retryOperationTimeout, reconcileDelay, func() error {
				resp, getErr := http.Get("http://localhost:7000/httpbin/get")
				if getErr != nil {
					t.Log(getErr)
					return getErr
				}

				if resp.StatusCode != 200 {
					t.Log("API is not created yet")
					return errors.New("API is not created yet")
				}

				return nil
			})

			is.NoErr(err)

			return ctx
		}).Feature()

	testenv.Test(t, opCreate)
}

func TestOperatorContextDelete(t *testing.T) {
	delApiDef := features.New("Operator Context Delete").
		Assess("Delete Api Defintion", func(ctx context.Context, t *testing.T, envConf *envconf.Config) context.Context {
			testNS := ctx.Value(ctxNSKey).(string)
			is := is.New(t)
			client := envConf.Client()

			// create operator context
			operatorCtx, err := createTestOperatorContext(ctx, testNS, envConf)
			is.NoErr(err) // failed to create operatorcontext

			// create api definition
			apiDef, err := createTestAPIDef(ctx, testNS, envConf)
			is.NoErr(err) // failed to create apiDefinition

			err = retryOperation(retryOperationTimeout, reconcileDelay, func() error {
				var opCtx v1alpha1.OperatorContext

				// shouldn't get deleted
				if errGet := client.Resources().Get(ctx, operatorCtx.Name, testNS, &opCtx); errGet != nil {
					t.Log(errGet)
					return errGet
				}

				if len(opCtx.Status.LinkedApiDefinitions) == 0 {
					t.Log("operator context status is not updated yet")
					return errors.New("operator context status is not updated yet")
				}

				t.Log("Operation completed successfully")

				return nil
			})
			is.NoErr(err)

			// try to delete operator context
			err = client.Resources().Delete(ctx, operatorCtx)
			is.NoErr(err)

			time.Sleep(reconcileDelay)

			var result v1alpha1.OperatorContext
			// shouldn't get deleted
			err = client.Resources().Get(ctx, operatorCtx.Name, testNS, &result)
			is.NoErr(err)

			// delete apidef
			err = client.Resources().Delete(ctx, apiDef)
			is.NoErr(err)

			err = retryOperation(retryOperationTimeout, reconcileDelay, func() error {
				var result v1alpha1.OperatorContext

				// should get deleted
				if errGet := client.Resources().Get(ctx, operatorCtx.Name, testNS, &result); errGet != nil {
					return nil
				}

				return errors.New("Should get deleted")
			})
			is.NoErr(err)

			return ctx
		}).Feature()

	updateApiDef := features.New("Operator Context Delete").
		Assess("Remove contextRef from Api Defintion", func(ctx context.Context, t *testing.T, envConf *envconf.Config) context.Context {
			testNS := ctx.Value(ctxNSKey).(string)
			is := is.New(t)
			client := envConf.Client()

			// create operator context
			operatorCtx, err := createTestOperatorContext(ctx, testNS, envConf)
			is.NoErr(err) // failed to create operatorcontext

			// create api definition
			apidef, err := createTestAPIDef(ctx, testNS, envConf)
			is.NoErr(err) // failed to create apiDefinition

			err = retryOperation(retryOperationTimeout, reconcileDelay, func() error {
				var opCtx v1alpha1.OperatorContext
				// shouldn't get deleted
				if errGet := client.Resources().Get(ctx, operatorCtx.Name, testNS, &opCtx); errGet != nil {
					return errGet
				}

				if len(opCtx.Status.LinkedApiDefinitions) == 0 {
					return errors.New("operator context status is not updated yet")
				}

				return nil
			})
			is.NoErr(err)

			// try to delete operator context
			err = client.Resources().Delete(ctx, operatorCtx)
			is.NoErr(err)

			time.Sleep(reconcileDelay)

			var result v1alpha1.OperatorContext
			// shouldn't get deleted
			err = client.Resources().Get(ctx, operatorCtx.Name, testNS, &result)
			is.NoErr(err)

			err = client.Resources().Get(ctx, apidef.Name, apidef.Namespace, apidef)
			is.NoErr(err)

			apidef.Spec.Context = nil

			err = client.Resources().Update(ctx, apidef)
			is.NoErr(err)

			err = retryOperation(retryOperationTimeout, reconcileDelay, func() error {
				var result v1alpha1.OperatorContext

				// should get deleted
				if err = client.Resources().Get(ctx, operatorCtx.Name, testNS, &result); err != nil {
					return nil
				}

				return errors.New("Should get deleted")
			})
			is.NoErr(err)

			return ctx
		}).Feature()

	testenv.Test(t, delApiDef)
	testenv.Test(t, updateApiDef)
}

func createTestAPIDef(ctx context.Context, namespace string, envConf *envconf.Config) (*v1alpha1.ApiDefinition, error) {
	var apiDef v1alpha1.ApiDefinition

	client := envConf.Client()

	apiDef.Name = "test-http"
	apiDef.Spec.Name = "test-http"
	apiDef.Namespace = namespace
	apiDef.Spec.Protocol = "http"
	apiDef.Spec.Context = &model.Target{
		Namespace: namespace,
		Name:      "mycontext",
	}
	apiDef.Spec.UseKeylessAccess = true
	apiDef.Spec.Active = true
	apiDef.Spec.Proxy = model.Proxy{
		ListenPath:      "/httpbin",
		TargetURL:       "http://httpbin.default.svc:8000",
		StripListenPath: true,
	}

	err := client.Resources(namespace).Create(ctx, &apiDef)

	return &apiDef, err
}

func createTestOperatorContext(ctx context.Context, namespace string, envConf *envconf.Config) (*v1alpha1.OperatorContext, error) {
	var operatorCtx v1alpha1.OperatorContext

	client := envConf.Client()

	operatorCtx.Name = "mycontext"
	operatorCtx.Namespace = namespace
	operatorCtx.Spec.FromSecret = &model.Target{
		Name:      "tyk-operator-conf",
		Namespace: operatorNamespace,
	}

	err := client.Resources(namespace).Create(ctx, &operatorCtx)

	return &operatorCtx, err
}
