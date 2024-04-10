/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import React from "react";
import { Plugin } from "../types/types";
import PluginTile from "./PluginTile";

interface PluginGridProps {
  plugins: Plugin[];
}

const PluginGrid: React.FC<PluginGridProps> = (props: PluginGridProps) => {
  return (
    <>
      <div className="org-info p-8 mb-8 bg-theme-background-lvl-0">
        <div className="grid grid-cols-[repeat(auto-fit,_minmax(20rem,_1fr))] auto-rows-[minmax(8rem,_1fr)] gap-6 pt-8">
          {props.plugins.map((plugin) => (
            <PluginTile key={plugin.metadata!.name!} plugin={plugin} />
          ))}
        </div>
      </div>
    </>
  );
};

export default PluginGrid;
