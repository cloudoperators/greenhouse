/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import {
  Container,
  DataGrid,
  DataGridCell,
  DataGridHeadCell,
  DataGridRow,
} from "juno-ui-components";

type OptionValues = {
  default?: unknown;
  description?: string | undefined;
  displayName?: string | undefined;
  name: string;
  regex?: string | undefined;
  required: boolean;
  type: "string" | "secret" | "bool" | "int" | "list" | "map";
}[];

interface OptionValueTableProps {
  optionValues: OptionValues;
}

const OptionValueTable: React.FC<OptionValueTableProps> = (
  props: OptionValueTableProps
) => {
  return (
    <Container px={false} py>
      <h2 className="text-xl font-bold mb-2 mt-8">Option Values</h2>
      <DataGrid columns={5}>
        <DataGridRow>
          <DataGridHeadCell>Name</DataGridHeadCell>
          <DataGridHeadCell>Required</DataGridHeadCell>
          <DataGridHeadCell>Description</DataGridHeadCell>
          <DataGridHeadCell>Type</DataGridHeadCell>
          <DataGridHeadCell>Default</DataGridHeadCell>
        </DataGridRow>
        {props.optionValues
          .sort((a, b) => {
            if ((a.required ?? false) && (!b.required ?? false)) {
              return -1;
            } else if ((!a.required ?? false) && (b.required ?? false)) {
              return 1;
            }
            return 0;
          })
          .map((option) => {
            return (
              <DataGridRow key={option.name}>
                <DataGridHeadCell>
                  {option.displayName ?? option.name}
                </DataGridHeadCell>
                <DataGridCell style={{ textAlign: "center" }}>
                  {(option.required ?? false) && "x"}
                </DataGridCell>
                <DataGridCell>{option.description}</DataGridCell>
                <DataGridCell>{option.type}</DataGridCell>
                <DataGridCell>
                  {option.default && JSON.stringify(option.default)}
                </DataGridCell>
              </DataGridRow>
            );
          })}
      </DataGrid>
    </Container>
  );
};

export default OptionValueTable;
