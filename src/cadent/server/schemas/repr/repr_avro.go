/*
Copyright 2014-2017 Bo Blanton

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*
   Avro defs for repr objects
*/

package repr

var AvroStatTag = `{
	"type":"record",
	"name":"Tag",
	"fields":
		[
			{"name":"Name", "type":"string"},
			{"name":"Value", "type":"string"}
		]
}`

/*
	message StatName {

	string key = 1;
	uint64 uniqueId = 2 [(gogoproto.customname) = "XXX_uniqueId", (gogoproto.jsontag) = "-"];
	string uniqueIdstr = 3 [(gogoproto.customname) = "XXX_uniqueIdstr", (gogoproto.jsontag) = "-"];
	uint32 resolution = 4;
	uint32 ttl = 5;
	TagMode tagMode = 6;
	repeated Tag tags = 13;
	repeated Tag meta_tags = 14;
}
*/
var AvroStatName = `{
	"type":"record",
	"name":"StatName",
	"fields":[
			{"name":"Key", "type": "string"},
			{"name":"Resolution", "type": ["null", "int"], "default": null}
			{"name":"Ttl", "type": ["null", "int"], "default": null}
			{"name":"TagMode", "type":"enum", "namespace": "cadent.repr", "symbols": ["METRICS2", "ALL"], "defaults": "ALL"},
			{
				"name": "Tags",
				"type": [
					"null",
					{
						"type": "array",
						"items": {
							"name": "Tag",
							"type": "record",
							"fields": [
								{"name": "Name", "type": "string"},
								{"name": "Value", "type": "string"}
							]
						}
					}
				],
				"default": null
			},
			{
				"name": "MetaTags",
				"type": [
					"null",
					{
						"type": "array",
						"items": {
							"name": "Tag",
							"type": "record",
							"fields": [
								{"name": "Name", "type": "string"},
								{"name": "Value", "type": "string"}
							]
						}
					}
				],
				"default": null
			}
		]
}`

/*
	StatName name = 1;
	int64 time = 2;
	double min = 3;
	double max = 4;
	double last = 5;
	double sum = 6;
	int64 count = 7;
*/
var AvroStatRepr = `{
	"type":"record",
	"name":"StatRepr",
	"fields":
		[
			"type":"record",
			"name": "Name",
			"namespace": "cadent.repr",
			"fields":[
					{"name":"Key", "type": "string"},
					{"name":"Resolution", "type": ["null", "int"], "default": null}
					{"name":"Ttl", "type": ["null", "int"], "default": null}
					{"name":"TagMode", "type":"enum", "namespace": "cadent.repr", "symbols": ["METRICS2", "ALL"], "defaults": "ALL"},
					{
						"name": "Tags",
						"type": [
							"null",
							{
								"type": "array",
								"items": {
									"name": "Tag",
									"type": "record",
									"fields": [
										{"name": "Name", "type": "string"},
										{"name": "Value", "type": "string"}
									]
								}
							}
						],
						"default": null
					},
					{
						"name": "MetaTags",
						"type": [
							"null",
							{
								"type": "array",
								"items": {
									"name": "Tag",
									"type": "record",
									"fields": [
										{"name": "Name", "type": "string"},
										{"name": "Value", "type": "string"}
									]
								}
							}
						],
						"default": null
					}
				],
			{"name": "Sum", "type": "double", "default": 0},
			{"name": "Min", "type": "double", "default": 0},
			{"name": "Max", "type": "double", "default": 0},
			{"name": "Last", "type": "double", "default": 0},
			{"name": "Count", "type": "long", "default": 0},

		]
}`
