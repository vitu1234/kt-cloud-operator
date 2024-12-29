package httpapi

import (
	v1beta1 "dcnlab.ssu.ac.kr/kt-cloud-operator/api/v1beta1"
)

func JoinControlPlane(ktMachineControlPlaneList []v1beta1.KTMachine, ktMachine v1beta1.KTMachine) error {

	var controlPlaneMachine v1beta1.KTMachine
	for i := 0; i < len(ktMachineControlPlaneList); i++ {
		err := CheckControlPlaneMachineReady(&ktMachineControlPlaneList[i])
		if err == nil {
			controlPlaneMachine = ktMachineControlPlaneList[i]
			break
		}
		logger1.Info("Control Plane not ready yet for " + ktMachineControlPlaneList[i].Name)
	}

	// we have to join the control plane

	return nil

}
