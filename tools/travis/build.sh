#!/bin/bash
set -e

# Build script for Travis-CI.

SCRIPTDIR=$(cd $(dirname "$0") && pwd)
ROOTDIR="$SCRIPTDIR/../.."
HOMEDIR="$SCRIPTDIR/../../../"

# clone the openwhisk utilities repo.
cd $HOMEDIR
git clone https://github.com/apache/incubator-openwhisk-utilities.git

# run the scancode util. against project source code starting at its root
incubator-openwhisk-utilities/scancode/scanCode.py $ROOTDIR --config $ROOTDIR/tools/build/scanCode.cfg

# run scalafmt checks
cd $ROOTDIR
TERM=dumb ./gradlew checkScalafmtAll

# lint tests to all be actually runnable
MISSING_TESTS=$(grep -rL "RunWith" --include="*Tests.scala" tests)
if [ -n "$MISSING_TESTS" ]
then
  echo "The following tests are missing the 'RunWith' annotation"
  echo $MISSING_TESTS
  exit 1
fi

cd $ROOTDIR/ansible

ANSIBLE_CMD="ansible-playbook -i environments/local -e docker_image_prefix=testing"

$ANSIBLE_CMD setup.yml -e mode=HA
$ANSIBLE_CMD prereq.yml
$ANSIBLE_CMD couchdb.yml
$ANSIBLE_CMD initdb.yml
$ANSIBLE_CMD apigateway.yml

cd $ROOTDIR

TERM=dumb ./gradlew distDocker -PdockerImagePrefix=testing 

cd $ROOTDIR/ansible

$ANSIBLE_CMD wipe.yml
$ANSIBLE_CMD openwhisk.yml

cd $ROOTDIR
cat whisk.properties
TERM=dumb ./gradlew :tests:testLean $GRADLE_PROJS_SKIP

cd $ROOTDIR/ansible
$ANSIBLE_CMD logs.yml

cd $ROOTDIR
tools/build/checkLogs.py logs
