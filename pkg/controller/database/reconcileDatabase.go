package database

import (
	"context"

	kciv1alpha1 "github.com/kloeckner-i/db-operator/pkg/apis/kci/v1alpha1"
	"github.com/kloeckner-i/db-operator/pkg/utils/database"
	"github.com/kloeckner-i/db-operator/pkg/utils/kci"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (r *ReconcileDatabase) getDatabaseSecret(dbcr *kciv1alpha1.Database) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	key := types.NamespacedName{
		Namespace: dbcr.Namespace,
		Name:      dbcr.Spec.SecretName,
	}
	err := r.client.Get(context.TODO(), key, secret)
	if err != nil {
		return nil, err
	}

	return secret, nil
}

func (r *ReconcileDatabase) getAdminSecret(dbcr *kciv1alpha1.Database) (*corev1.Secret, error) {
	instance, err := dbcr.GetInstanceRef()
	if err != nil {
		// failed to get DbInstanceRef this case should not happen
		return nil, err
	}

	// get database admin credentials
	secret := &corev1.Secret{}

	err = r.client.Get(context.TODO(), instance.Spec.AdminUserSecret, secret)
	if err != nil {
		return nil, err
	}

	return secret, nil
}

// createDatabase secret, actual database using admin secret
func (r *ReconcileDatabase) createDatabase(dbcr *kciv1alpha1.Database) error {
	databaseSecret, err := r.getDatabaseSecret(dbcr)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			secretData, err := generateDatabaseSecretData(dbcr)
			if err != nil {
				logrus.Errorf("can not generate credentials for database - %s", err)
				return err
			}
			newDatabaseSecret := &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      dbcr.Spec.SecretName,
					Namespace: dbcr.Namespace,
					Labels:    kci.BaseLabelBuilder(),
				},
				Data: secretData,
			}
			err = r.client.Create(context.TODO(), newDatabaseSecret)
			if err != nil {
				// failed to create secret
				return err
			}
			databaseSecret = newDatabaseSecret
		} else {
			// failed to get secret resouce
			return err
		}
	}

	databaseCred, err := parseDatabaseSecretData(dbcr, databaseSecret.Data)
	if err != nil {
		// failed to parse database credential from secret
		return err
	}

	db, err := determinDatabaseType(dbcr, databaseCred)
	if err != nil {
		// failed to determine database type
		return err
	}

	adminSecretResource, err := r.getAdminSecret(dbcr)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			logrus.Errorf("can not find admin secret")
			return err
		}
		return err
	}

	// found admin secret. parse it to connect database
	adminCred, err := db.ParseAdminCredentials(adminSecretResource.Data)
	if err != nil {
		// failed to parse database admin secret
		return err
	}

	err = database.Create(db, adminCred)
	if err != nil {
		logrus.Errorf("DB: namespace=%s, name=%s failed creating database", dbcr.Namespace, dbcr.Name)
		return err
	}

	instance, err := dbcr.GetInstanceRef()
	if err != nil {
		return err
	}

	databaseConfigResource := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: dbcr.Namespace,
			Name:      dbcr.Spec.SecretName,
			Labels:    kci.BaseLabelBuilder(),
		},
		Data: instance.Status.Info,
	}

	err = r.client.Create(context.TODO(), databaseConfigResource)
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			// if configmap resource already exists, update
			err = r.client.Update(context.TODO(), databaseConfigResource)
			if err != nil {
				logrus.Errorf("DB: namespace=%s, name=%s failed updating database info configmap", dbcr.Namespace, dbcr.Name)
				return err
			}
		} else {
			logrus.Errorf("DB: namespace=%s, name=%s failed creating database info configmap", dbcr.Namespace, dbcr.Name)
			return err
		}
	}

	logrus.Infof("DB: namespace=%s, name=%s successfully created", dbcr.Namespace, dbcr.Name)
	return nil
}

func (r *ReconcileDatabase) deleteDatabase(dbcr *kciv1alpha1.Database) error {
	if dbcr.Spec.DeletionProtected {
		logrus.Infof("DB: namespace=%s, name=%s is deletion protected. will not be deleted in backends", dbcr.Name, dbcr.Namespace)
		return nil
	}

	// Todo: save database in info and use it for deletion, instead of re-calculating dbname for deletion info
	secretData, err := generateDatabaseSecretData(dbcr)
	if err != nil {
		logrus.Errorf("can not generate credentials for database - %s", err)
		return err
	}

	databaseCred, err := parseDatabaseSecretData(dbcr, secretData)
	if err != nil {
		// failed to parse database credential from secret
		return err
	}

	db, err := determinDatabaseType(dbcr, databaseCred)
	if err != nil {
		// failed to determine database type
		return err
	}

	adminSecretResource, err := r.getAdminSecret(dbcr)
	if err != nil {
		// failed to get admin secret
		return err
	}
	// found admin secret. parse it to connect database
	adminCred, err := db.ParseAdminCredentials(adminSecretResource.Data)
	if err != nil {
		// failed to parse database admin secret
		return err
	}

	err = database.Delete(db, adminCred)
	if err != nil {
		return err
	}

	return nil
}
