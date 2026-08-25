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

	// Flow nodes
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

	// Connectivity & Zeebe extensions
	"sequenceFlow",
	"messageFlow",
	"extensionElements",
	"conditionExpression",
}
