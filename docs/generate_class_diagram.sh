# Copyright 2020 Red Hat, Inc
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

pushd ..
goplantuml -recursive . > class_diagram.uml
java -jar ~/tools/plantuml.jar class_diagram.uml
java -jar ~/tools/plantuml.jar -tsvg class_diagram.uml
xmllint --format class_diagram.svg > class_diagram_.svg
mv class_diagram.uml docs/
mv class_diagram.svg docs/images/
mv class_diagram_.svg docs/images/
mv class_diagram.png docs/images/
popd
