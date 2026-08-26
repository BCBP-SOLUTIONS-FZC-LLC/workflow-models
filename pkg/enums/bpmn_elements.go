package enums

var AllowedBPMNElements = []string{
	// Root structure
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

	// Flow nodes
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

	// Boundary / intermediate events
	"boundaryEvent",
	"intermediateCatchEvent",
	"timerEventDefinition",
	"errorEventDefinition",
	"messageEventDefinition",
	"timeDuration",

	// Connectivity & Zeebe extensions
	"sequenceFlow",
	"messageFlow",
	"extensionElements",
	"conditionExpression",
	"incoming",
	"outgoing",
}
