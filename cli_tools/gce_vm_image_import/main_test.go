//  Copyright 2018 Google Inc. All Rights Reserved.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package main

import (
	"fmt"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"google.golang.org/api/compute/v1"
	"os"
	"testing"
)

func TestGetRegion(t *testing.T) {
	tests := []struct {
		input string
		want  string
		err   error
	}{
		{"us-central1-c", "us-central1", nil},
		{"europe-north1-a", "europe-north1", nil},
		{"europe", "", fmt.Errorf("%v is not a valid zone", "europe")},
		{"", "", fmt.Errorf("zone is empty. Can't determine region")},
	}

	oldZone := zone
	for _, test := range tests {
		zone = &test.input
		got, err := getRegion()
		if test.want != got {
			t.Errorf("%v != %v", test.want, got)
		} else if err != test.err && test.err.Error() != err.Error() {
			t.Errorf("%v != %v", test.err, err)
		}
	}
	zone = oldZone
}

func TestPopulateRegion(t *testing.T) {
	tests := []struct {
		input string
		want  string
		err   error
	}{
		{"us-central1-c", "us-central1", nil},
		{"europe", "", fmt.Errorf("%v is not a valid zone", "europe")},
		{"", "", fmt.Errorf("zone is empty. Can't determine region")},
	}

	oldZone := zone
	for _, test := range tests {
		zone = &test.input
		region = nil
		err := populateRegion()
		if err != test.err && test.err.Error() != err.Error() {
			t.Errorf("%v != %v", test.err, err)
		} else if region != nil && test.want != *region {
			t.Errorf("%v != %v", test.want, *region)
		}
	}
	zone = oldZone
}

func TestGetWorkflowPathsFromImage(t *testing.T) {
	defer setStringP(&sourceImage, "image-1")()
	defer setStringP(&osID, "ubuntu-1404")()
	workflow, translate := getWorkflowPaths()
	if workflow != importFromImageWorkflow && translate != "ubuntu/translate_ubuntu_1404.wf.json" {
		t.Errorf("%v != %v and/or translate not empty", workflow, importFromImageWorkflow)
	}
}

func TestGetWorkflowPathsDataDisk(t *testing.T) {
	defer setBoolP(&dataDisk, true)()
	workflow, translate := getWorkflowPaths()
	if workflow != importWorkflow && translate != "" {
		t.Errorf("%v != %v and/or translate not empty", workflow, importWorkflow)
	}
}

func TestGetWorkflowPathsFromFile(t *testing.T) {
	defer setBoolP(&dataDisk, false)()
	defer setStringP(&sourceImage, "image-1")()
	defer setStringP(&osID, "ubuntu-1404")()
	defer setStringP(&sourceImage, "")()

	workflow, translate := getWorkflowPaths()

	if workflow != importAndTranslateWorkflow || translate != "ubuntu/translate_ubuntu_1404.wf.json" {
		t.Errorf("%v != %v and/or translate not %v", workflow, "ubuntu/translate_ubuntu_1404.wf.json", importAndTranslateWorkflow)
	}
}

func TestFlagsImageNameNotProvided(t *testing.T) {
	err := validateAndParseFlags()
	expected := fmt.Errorf("The flag -image_name must be provided")
	if err != expected && err.Error() != expected.Error() {
		t.Errorf("%v != %v", err, expected)
	}
}

func TestFlagsClientIdNotProvided(t *testing.T) {
	defer backupOsArgs()()
	cliArgs := getAllCliArgs()
	defer clearStringFlag(cliArgs, clientIDFlagKey, &clientID)()
	buildOsArgsAndAssertErrorOnValidate(cliArgs, "Expected error for missing client_id flag", t)
}

func TestFlagsDataDiskOrOSFlagsNotProvided(t *testing.T) {
	defer backupOsArgs()()
	cliArgs := getAllCliArgs()
	defer clearStringFlag(cliArgs, "os", &osID)()
	defer clearBoolFlag(cliArgs, "data_disk", &dataDisk)()
	buildOsArgsAndAssertErrorOnValidate(cliArgs, "Expected error for missing os or data_disk flag", t)
}

func TestFlagsDataDiskAndOSFlagsBothProvided(t *testing.T) {
	defer backupOsArgs()()
	cliArgs := getAllCliArgs()
	buildOsArgsAndAssertErrorOnValidate(cliArgs, "Expected error for both os and data_disk set at the same time", t)
}

func TestFlagsSourceFileOrSourceImageNotProvided(t *testing.T) {
	defer backupOsArgs()()
	cliArgs := getAllCliArgs()
	defer clearStringFlag(cliArgs, "source_file", &sourceFile)()
	defer clearStringFlag(cliArgs, "source_image", &sourceImage)()
	defer clearBoolFlag(cliArgs, "data_disk", &dataDisk)()
	buildOsArgsAndAssertErrorOnValidate(cliArgs, "Expected error for missing source_file or source_image flag", t)
}

func TestFlagsSourceFileAndSourceImageBothProvided(t *testing.T) {
	defer backupOsArgs()()
	cliArgs := getAllCliArgs()
	defer clearBoolFlag(cliArgs, "data_disk", &dataDisk)()
	buildOsArgsAndAssertErrorOnValidate(cliArgs, "Expected error for both source_file and source_image flags set", t)
}

func buildOsArgsAndAssertErrorOnValidate(cliArgs map[string]interface{}, errorMsg string, t *testing.T) {
	buildOsArgs(cliArgs)
	if err := validateAndParseFlags(); err == nil {
		t.Error(errorMsg)
	}
}

func TestFlagsSourceFile(t *testing.T) {
	defer backupOsArgs()()

	cliArgs := getAllCliArgs()
	defer clearStringFlag(cliArgs, "source_image", &sourceImage)()
	defer clearBoolFlag(cliArgs, "data_disk", &dataDisk)()
	buildOsArgs(cliArgs)

	if err := validateAndParseFlags(); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestFlagsInvalidSourceFile(t *testing.T) {
	defer backupOsArgs()()

	cliArgs := getAllCliArgs()
	cliArgs["source_file"] = "invalidSourceFile"
	defer clearStringFlag(cliArgs, "source_image", &sourceImage)()
	defer clearBoolFlag(cliArgs, "data_disk", &dataDisk)()
	buildOsArgs(cliArgs)

	if err := validateAndParseFlags(); err == nil {
		t.Errorf("Expected error")
	}
}

func TestFlagsSourceImage(t *testing.T) {
	defer backupOsArgs()()

	cliArgs := getAllCliArgs()
	defer clearStringFlag(cliArgs, "source_file", &sourceFile)()
	defer clearBoolFlag(cliArgs, "data_disk", &dataDisk)()
	buildOsArgs(cliArgs)

	if err := validateAndParseFlags(); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestFlagsDataDisk(t *testing.T) {
	defer backupOsArgs()()

	cliArgs := getAllCliArgs()
	defer clearStringFlag(cliArgs, "source_image", &sourceImage)()
	defer clearStringFlag(cliArgs, "os", &osID)()
	buildOsArgs(cliArgs)

	if err := validateAndParseFlags(); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestFlagsInvalidOS(t *testing.T) {
	defer backupOsArgs()()

	cliArgs := getAllCliArgs()
	defer clearBoolFlag(cliArgs, "data_disk", &dataDisk)()
	defer clearStringFlag(cliArgs, "source_image", &sourceImage)()
	cliArgs["os"] = "invalidOs"
	buildOsArgs(cliArgs)

	if err := validateAndParseFlags(); err == nil {
		t.Errorf("Expected error")
	}
}

func TestUpdateWorkflowInstancesLabelled(t *testing.T) {
	defer setBoolP(&noExternalIP, false)()
	buildID = "abc"

	existingLabels := map[string]string{"labelKey": "labelValue"}
	w := daisy.New()
	w.Steps = map[string]*daisy.Step{
		"ci": {
			CreateInstances: &daisy.CreateInstances{
				{
					Instance: compute.Instance{
						Disks:  []*compute.AttachedDisk{{Source: "key1"}},
						Labels: map[string]string{"labelKey": "labelValue"},
					},
				},
				{
					Instance: compute.Instance{
						Disks: []*compute.AttachedDisk{{Source: "key2"}},
					},
				},
			},
		},
	}

	updateWorkflow(w)
	validateLabels(&(*w.Steps["ci"].CreateInstances)[0].Instance.Labels, "gce-image-import-tmp", t, &existingLabels)
	validateLabels(&(*w.Steps["ci"].CreateInstances)[1].Instance.Labels, "gce-image-import-tmp", t)
}

func TestUpdateWorkflowDisksLabelled(t *testing.T) {
	defer setBoolP(&noExternalIP, false)()
	buildID = "abc"

	extraLabels := map[string]string{"labelKey": "labelValue"}
	w := createWorkflowWithCreateDisksStep()

	updateWorkflow(w)
	validateLabels(&(*w.Steps["cd"].CreateDisks)[0].Disk.Labels, "gce-image-import-tmp", t, &extraLabels)
	validateLabels(&(*w.Steps["cd"].CreateDisks)[1].Disk.Labels, "gce-image-import-tmp", t)
}

func createWorkflowWithCreateDisksStep() *daisy.Workflow {
	w := daisy.New()
	w.Steps = map[string]*daisy.Step{
		"cd": {
			CreateDisks: &daisy.CreateDisks{
				{
					Disk: compute.Disk{
						Labels: map[string]string{"labelKey": "labelValue"},
					},
				},
				{
					Disk: compute.Disk{},
				},
			},
		},
	}
	return w
}

func TestUpdateWorkflowIncludedWorkflow(t *testing.T) {
	defer setBoolP(&noExternalIP, false)()
	buildID = "abc"

	childWorkflow := createWorkflowWithCreateDisksStep()
	extraLabels := map[string]string{"labelKey": "labelValue"}

	w := daisy.New()
	w.Steps = map[string]*daisy.Step{
		"cd": {
			IncludeWorkflow: &daisy.IncludeWorkflow{
				Workflow: childWorkflow,
			},
		},
	}

	updateWorkflow(w)
	validateLabels(&(*childWorkflow.Steps["cd"].CreateDisks)[0].Disk.Labels, "gce-image-import-tmp", t, &extraLabels)
	validateLabels(&(*childWorkflow.Steps["cd"].CreateDisks)[1].Disk.Labels, "gce-image-import-tmp", t)
}

func TestUpdateWorkflowImagesLabelled(t *testing.T) {
	defer setBoolP(&noExternalIP, false)()
	buildID = "abc"

	w := daisy.New()
	extraLabels := map[string]string{"labelKey": "labelValue"}
	w.Steps = map[string]*daisy.Step{
		"cimg": {
			CreateImages: &daisy.CreateImages{
				{
					Image: compute.Image{
						Name:   "final-image-1",
						Labels: map[string]string{"labelKey": "labelValue"},
					},
				},
				{
					Image: compute.Image{
						Name: "final-image-2",
					},
				},
				{
					Image: compute.Image{
						Name:   "untranslated-image-1",
						Labels: map[string]string{"labelKey": "labelValue"},
					},
				},
				{
					Image: compute.Image{
						Name: "untranslated-image-2",
					},
				},
			},
		},
	}

	updateWorkflow(w)

	validateLabels(&(*w.Steps["cimg"].CreateImages)[0].Image.Labels, "gce-image-import", t, &extraLabels)
	validateLabels(&(*w.Steps["cimg"].CreateImages)[1].Image.Labels, "gce-image-import", t)

	validateLabels(&(*w.Steps["cimg"].CreateImages)[2].Image.Labels, "gce-image-import-tmp", t, &extraLabels)
	validateLabels(&(*w.Steps["cimg"].CreateImages)[3].Image.Labels, "gce-image-import-tmp", t)
}

func TestUpdateWorkflowInstancesConfiguredForNoExternalIP(t *testing.T) {
	defer setBoolP(&noExternalIP, true)()

	w := createWorkflowWithCreateInstanceNetworkAccessConfig()
	updateWorkflow(w)

	if len((*w.Steps["ci"].CreateInstances)[0].Instance.NetworkInterfaces[0].AccessConfigs) != 0 {
		t.Errorf("Instance AccessConfigs not empty")
	}
}

func TestUpdateWorkflowInstancesNotModifiedIfExternalIPAllowed(t *testing.T) {
	defer setBoolP(&noExternalIP, false)()

	w := createWorkflowWithCreateInstanceNetworkAccessConfig()
	updateWorkflow(w)

	if len((*w.Steps["ci"].CreateInstances)[0].Instance.NetworkInterfaces[0].AccessConfigs) != 1 {
		t.Errorf("Instance AccessConfigs doesn't have exactly one instance")
	}
}

func TestUpdateWorkflowInstancesNotModifiedIfNoNetworkInterfaceElement(t *testing.T) {
	defer setBoolP(&noExternalIP, true)()
	w := createWorkflowWithCreateInstanceNetworkAccessConfig()
	(*w.Steps["ci"].CreateInstances)[0].Instance.NetworkInterfaces = nil
	updateWorkflow(w)

	if (*w.Steps["ci"].CreateInstances)[0].Instance.NetworkInterfaces != nil {
		t.Errorf("Instance NetworkInterfaces should stay nil if nil before update")
	}
}

func TestBuildDaisyVarsFromDisk(t *testing.T) {
	defer setStringP(&imageName, "image-a")()
	defer setBoolP(&noGuestEnvironment, true)()
	defer setStringP(&sourceFile, "source-file-path")()
	defer setStringP(&sourceImage, "")()
	defer setStringP(&family, "a-family")()
	defer setStringP(&description, "a-description")()
	defer setStringP(&network, "a-network")()
	defer setStringP(&subnet, "a-subnet")()
	defer setStringP(&region, "a-region")()

	got := buildDaisyVars("translate/workflow/path")

	assertEqual(got["image_name"], "image-a", t)
	assertEqual(got["translate_workflow"], "translate/workflow/path", t)
	assertEqual(got["install_gce_packages"], "false", t)
	assertEqual(got["source_disk_file"], "source-file-path", t)
	assertEqual(got["family"], "a-family", t)
	assertEqual(got["description"], "a-description", t)
	assertEqual(got["import_network"], "global/networks/a-network", t)
	assertEqual(got["import_subnet"], "regions/a-region/subnetworks/a-subnet", t)
	assertEqual(len(got), 8, t)
}

func TestBuildDaisyVarsFromImage(t *testing.T) {
	defer setStringP(&imageName, "image-a")()
	defer setBoolP(&noGuestEnvironment, true)()
	defer setStringP(&sourceFile, "")()
	defer setStringP(&sourceImage, "source-image")()
	defer setStringP(&family, "a-family")()
	defer setStringP(&description, "a-description")()
	defer setStringP(&network, "a-network")()
	defer setStringP(&subnet, "a-subnet")()
	defer setStringP(&region, "a-region")()

	got := buildDaisyVars("translate/workflow/path")

	assertEqual(got["image_name"], "image-a", t)
	assertEqual(got["translate_workflow"], "translate/workflow/path", t)
	assertEqual(got["install_gce_packages"], "false", t)
	assertEqual(got["source_image"], "global/images/source-image", t)
	assertEqual(got["family"], "a-family", t)
	assertEqual(got["description"], "a-description", t)
	assertEqual(got["import_network"], "global/networks/a-network", t)
	assertEqual(got["import_subnet"], "regions/a-region/subnetworks/a-subnet", t)
	assertEqual(len(got), 8, t)
}

var onGCE *bool
var gceZone *string
var gceMetadataError error

func TestPopulateZoneIfMissingOnGCESuccess(t *testing.T) {
	defer setBoolP(&onGCE, true)()
	defer setStringP(&gceZone, "europe-north1-c")()
	defer setStringP(&zone, "")()
	err := populateZoneIfMissing(dummyMetadataGCE{})

	assertNil(err, t)
	assertEqual(*zone, "europe-north1-c", t)
}

func TestPopulateZoneIfMissingNotOnGCEZoneDoesntChange(t *testing.T) {
	defer setBoolP(&onGCE, false)()
	defer setStringP(&zone, "us-west-1c")()
	defer setStringP(&gceZone, "europe-north1-c")()

	err := populateZoneIfMissing(dummyMetadataGCE{})

	assertNil(err, t)
	assertEqual(*zone, "us-west-1c", t)
}

func TestPopulateZoneIfMissingOnGCEError(t *testing.T) {
	defer setBoolP(&onGCE, true)()
	defer setStringP(&gceZone, "europe-north1-c")()
	defer setStringP(&zone, "")()
	gceMetadataError = fmt.Errorf("zone error")

	assertError(populateZoneIfMissing(dummyMetadataGCE{}), t)

	gceMetadataError = nil
}

func TestPopulateZoneIfMissingOnGCEEnmptyZoneReturned(t *testing.T) {
	defer setBoolP(&onGCE, true)()
	defer setStringP(&gceZone, "")()
	defer setStringP(&zone, "")()
	assertError(populateZoneIfMissing(dummyMetadataGCE{}), t)
}

func TestParseUserLabelsReturnsEmptyMap(t *testing.T) {
	labels := ""
	labelsMap, err := parseUserLabels(&labels)
	assertNil(err, t)
	assertNotNil(labelsMap, t)
	if len(*labelsMap) > 0 {
		t.Errorf("Labels map %v should be empty, but it's size is %v.", labelsMap, len(*labelsMap))
	}
}

func TestParseUserLabels(t *testing.T) {
	doTestParseUserLabels("userkey1=uservalue1,userkey2=uservalue2", t)
}

func TestParseUserLabelsWithWhiteChars(t *testing.T) {
	doTestParseUserLabels("	 userkey1 = uservalue1	,	 userkey2	 =	 uservalue2	 ", t)
}

func doTestParseUserLabels(labels string, t *testing.T) {
	labelsMap, err := parseUserLabels(&labels)
	assertNil(err, t)
	assertNotNil(labelsMap, t)
	if len(*labelsMap) != 2 {
		t.Errorf("Labels map %v should be have size 2, but it's %v.", labelsMap, len(*labelsMap))
	}
	assertEqual("uservalue1", (*labelsMap)["userkey1"], t)
	assertEqual("uservalue2", (*labelsMap)["userkey2"], t)
}

func TestParseUserLabelsNoEqualsSignSingleLabel(t *testing.T) {
	doTestParseUserLabelsNoEqualsSign("userkey1", t)
}

func TestParseUserLabelsNoEqualsSignLast(t *testing.T) {
	doTestParseUserLabelsNoEqualsSign("userkey2=uservalue2,userkey1", t)
}

func TestParseUserLabelsNoEqualsSignFirst(t *testing.T) {
	doTestParseUserLabelsNoEqualsSign("userkey1,userkey2=uservalue2", t)
}

func TestParseUserLabelsNoEqualsSignMiddle(t *testing.T) {
	doTestParseUserLabelsNoEqualsSign("userkey3=uservalue3,userkey1,userkey2=uservalue2", t)
}

func TestParseUserLabelsNoEqualsSignMultiple(t *testing.T) {
	doTestParseUserLabelsNoEqualsSign("userkey1,userkey2", t)
}

func TestParseUserLabelsWhiteSpacesOnlyInKey(t *testing.T) {
	doTestParseUserLabelsError(" 	=uservalue1", "Labels map %v should be nil for %v", t)
}

func TestParseUserLabelsWhiteSpacesOnlyInValue(t *testing.T) {
	doTestParseUserLabelsError("userkey= 	", "Labels map %v should be nil for %v", t)
}

func TestParseUserLabelsWhiteSpacesOnlyInKeyAndValue(t *testing.T) {
	doTestParseUserLabelsError(" 	= 	", "Labels map %v should be nil for %v", t)
}

func doTestParseUserLabelsNoEqualsSign(labels string, t *testing.T) {
	doTestParseUserLabelsError(labels, "Labels map %v should be nil for v%", t)
}

func doTestParseUserLabelsError(labels string, errorMsg string, t *testing.T) {
	labelsMap, err := parseUserLabels(&labels)
	if labelsMap != nil {
		t.Errorf(errorMsg, labelsMap, labels)
	}
	assertNotNil(err, t)
}

type dummyMetadataGCE struct{}

func (m dummyMetadataGCE) OnGCE() bool {
	return *onGCE
}

func (m dummyMetadataGCE) Zone() (string, error) {
	return *gceZone, gceMetadataError
}

func assertEqual(i1 interface{}, i2 interface{}, t *testing.T) {
	if i1 != i2 {
		t.Errorf("%v != %v", i1, i2)
	}
}

func assertNotNil(i1 interface{}, t *testing.T) {
	if i1 == nil {
		t.Errorf("%v is nil", i1)
	}
}

func assertNil(i1 interface{}, t *testing.T) {
	if i1 != nil {
		t.Errorf("%v not nil", i1)
	}
}

func assertError(err error, t *testing.T) {
	if err == nil {
		t.Errorf("%v error is empty", err)
	}
}

func createWorkflowWithCreateInstanceNetworkAccessConfig() *daisy.Workflow {
	w := daisy.New()
	w.Steps = map[string]*daisy.Step{
		"ci": {
			CreateInstances: &daisy.CreateInstances{
				{
					Instance: compute.Instance{
						Disks: []*compute.AttachedDisk{{Source: "key1"}},
						NetworkInterfaces: []*compute.NetworkInterface{
							{
								Network: "n",
								AccessConfigs: []*compute.AccessConfig{
									{Type: "ONE_TO_ONE_NAT"},
								},
							},
						},
					},
				},
			},
		},
	}
	return w
}

func validateLabels(labels *map[string]string, typeLabel string, t *testing.T, extraLabelsArr ...*map[string]string) {
	var extraLabels *map[string]string
	if len(extraLabelsArr) > 0 {
		extraLabels = extraLabelsArr[0]
	}
	extraLabelCount := 0
	if extraLabels != nil {
		extraLabelCount = len(*extraLabels)
	}
	expectedLabelCount := extraLabelCount + 4
	if len(*labels) != expectedLabelCount {
		t.Errorf("Labels %v should have only %v elements", labels, expectedLabelCount)
	}
	assertEqual(buildID, (*labels)["gce-image-import-build-id"], t)
	assertEqual("true", (*labels)[typeLabel], t)
	assertEqual("uservalue1", (*labels)["userkey1"], t)
	assertEqual("uservalue2", (*labels)["userkey2"], t)

	if extraLabels != nil {
		for extraKey, extraValue := range *extraLabels {
			if value, ok := (*labels)[extraKey]; !ok || value != extraValue {
				t.Errorf("Key %v from labels missing or value %v not matching", extraKey, value)
			}
		}
	}
}

func backupOsArgs() func() {
	oldArgs := os.Args
	return func() { os.Args = oldArgs }
}

func buildOsArgs(cliArgs map[string]interface{}) {
	os.Args = make([]string, len(cliArgs)+1)
	i := 0
	os.Args[i] = "cmd"
	i++
	for key, value := range cliArgs {
		if value != nil {
			os.Args[i] = formatCliArg(key, value)
			i++
		}
	}
}

func formatCliArg(argKey, argValue interface{}) string {
	if argValue == true {
		return fmt.Sprintf("-%v", argKey)
	}
	if argValue != false {
		return fmt.Sprintf("-%v=%v", argKey, argValue)
	}
	return ""
}

func getAllCliArgs() map[string]interface{} {
	return map[string]interface{}{
		imageNameFlagKey:            "img",
		clientIDFlagKey:             "aClient",
		"data_disk":                 true,
		"os":                        "ubuntu-1404",
		"source_file":               "gs://source_bucket/source_file",
		"source_image":              "anImage",
		"no_guest_environment":      true,
		"family":                    "aFamily",
		"description":               "aDescription",
		"network":                   "aNetwork",
		"subnet":                    "aSubnet",
		"timeout":                   "2h",
		"zone":                      "us-central1-c",
		"project":                   "aProject",
		"scratch_bucket_gcs_path":   "gs://bucket/folder",
		"oauth":                     "oAuthFilePath",
		"compute_endpoint_override": "us-east1-c",
		"disable_gcs_logging":       true,
		"disable_cloud_logging":     true,
		"disable_stdout_logging":    true,
		"kms_key":                   "aKmsKey",
		"kms_keyring":               "aKmsKeyRing",
		"kms_location":              "aKmsLocation",
		"kms_project":               "aKmsProject",
		"labels":                    "userkey1=uservalue1,userkey2=uservalue2",
	}
}

func setStringP(p **string, value string) func() {
	oldValue := *p
	*p = &value
	return func() {
		*p = oldValue
	}
}

func setBoolP(p **bool, value bool) func() {
	oldValue := *p
	*p = &value
	return func() { *p = oldValue }
}

func clearStringFlag(cliArgs map[string]interface{}, flagKey string, flag **string) func() {
	delete(cliArgs, flagKey)
	return setStringP(flag, "")
}

func clearBoolFlag(cliArgs map[string]interface{}, flagKey string, flag **bool) func() {
	delete(cliArgs, flagKey)
	return setBoolP(flag, false)
}
