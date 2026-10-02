package enums

// AllowedBPMNElements is the Tier-1 allowlist of BPMN XML element names
// Definition Service's compiler enforces; the modeler-facing
// GET /bpmn/allowed-elements endpoint serves the same list to drive the
// authoring UI's palette (definition LLD §4.1.2, §3.3.20).
// An element name absent from this list is rejected by the compiler, not
// silently passed through. The blank lines below group entries by role
// (collaboration/process structure and the diagram-only decorations a modeler
// draws, task/gateway/event types, boundary and
// intermediate event definitions, flow/expression wiring) for readability
// only — membership, not position, is what's enforced.
var AllowedBPMNElements = []string{
	"definitions",
	"collaboration",
	"participant",
	"process",
	"laneSet",
	"lane",
	"flowNodeRef",
	"error",
	"message",
	"dataStoreReference",
	"dataInputAssociation",
	"dataOutputAssociation",
	"sourceRef",
	"targetRef",
	"textAnnotation",
	"text",
	"association",
	"property",
	"documentation",

	"task",
	"startEvent",
	"endEvent",
	"userTask",
	"sendTask",
	"receiveTask",
	"serviceTask",
	"parallelGateway",
	"exclusiveGateway",
	"inclusiveGateway",
	"eventBasedGateway",
	"subProcess",
	"callActivity",

	"boundaryEvent",
	"intermediateCatchEvent",
	"timerEventDefinition",
	"errorEventDefinition",
	"messageEventDefinition",
	"timeDuration",

	"sequenceFlow",
	"messageFlow",
	"extensionElements",
	"conditionExpression",
	"incoming",
	"outgoing",
}
