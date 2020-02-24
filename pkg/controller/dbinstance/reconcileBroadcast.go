package dbinstance

import (
	"context"
	kciv1alpha1 "github.com/kloeckner-i/db-operator/pkg/apis/kci/v1alpha1"
)

func (r *ReconcileDbInstance) broadcast(dbin *kciv1alpha1.DbInstance) error {
	dbList := &kciv1alpha1.DatabaseList{}
	err := r.client.List(context.Background(), dbList)
	if err != nil {
		return err
	}

	for _, db := range dbList.Items {
		ref, err := db.GetInstanceRef()
		if err != nil {
			continue
		}
		if ref.Name == dbin.Name {
			annotations := db.ObjectMeta.GetAnnotations()
			if _, found := annotations["checksum/spec"]; found {
				annotations["checksum/spec"] = ""
				db.ObjectMeta.SetAnnotations(annotations)
				err = r.client.Update(context.Background(), &db)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
