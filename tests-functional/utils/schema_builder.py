import json
import os

from genson import SchemaBuilder

from conftest import option


class CustomSchemaBuilder(SchemaBuilder):

    def __init__(self, schema_name):
        super().__init__()
        self.path = f"{option.base_dir}/schemas/{schema_name}"

    def create_schema(self, response_json):
        builder = SchemaBuilder()
        builder.add_object(response_json)
        schema = builder.to_schema()

        os.makedirs(os.path.dirname(self.path), exist_ok=True)
        with open(self.path, "w") as file:
            json.dump(schema, file, sort_keys=True, indent=4, ensure_ascii=False)
