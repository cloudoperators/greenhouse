/*
 * SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
 * SPDX-License-Identifier: Apache-2.0
 */

import { Icon, Stack } from "juno-ui-components";
import React from "react";
import { Plugin } from "../types/types";
import useStore from "./../store";

interface PluginTileProps {
  plugin: Plugin;
}
const allowedFileEndings = [".png", ".jpg", ".jpeg"];
const PluginTile: React.FC<PluginTileProps> = (props: PluginTileProps) => {
  const setShowPluginDetails = useStore((state) => state.setShowPluginDetails);
  const setPluginDetail = useStore((state) => state.setPluginDetail);

  let iconUrl: string | undefined;
  if (
    allowedFileEndings.some((ending) =>
      props.plugin.spec?.icon?.endsWith(ending)
    )
  ) {
    iconUrl = props.plugin.spec?.icon;
  } else {
    iconUrl = undefined;
  }

  const openPluginDetails = () => {
    setShowPluginDetails(true);
    setPluginDetail(props.plugin);
  };
  return (
    <Stack
      direction="vertical"
      alignment="center"
      distribution="between"
      className="org-info-item bg-theme-background-lvl-1 p-4"
      style={{ cursor: "pointer" }}
      onClick={openPluginDetails}
    >
      <h2 className="text-lg font-bold">
        {props.plugin.spec?.displayName ?? props.plugin.metadata?.name}
      </h2>

      {!iconUrl && (
        <Icon
          icon={props.plugin.spec?.icon ?? "autoAwesomeMosaic"}
          size="100"
        />
      )}
      {iconUrl && (
        <img
          className="filtered"
          src={iconUrl}
          alt="icon"
          width="100"
          height="100"
        />
      )}
      <p>{props.plugin.spec?.description}</p>

      <div className="bg-theme-background-lvl-4 py-2 px-3 inline-flex">
        {props.plugin.spec?.version}
      </div>
    </Stack>
  );
};

export default PluginTile;
