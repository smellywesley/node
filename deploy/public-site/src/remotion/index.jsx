import React from "react";
import { Composition, registerRoot } from "remotion";
import { hyperframes } from "../hyperframes.js";
import { NodeSignalPlane } from "./node-signal-plane.jsx";

function RemotionRoot() {
  return (
    <Composition
      id="NodeSignalPlane"
      component={NodeSignalPlane}
      durationInFrames={hyperframes.durationInFrames}
      fps={hyperframes.fps}
      width={hyperframes.width}
      height={hyperframes.height}
      defaultProps={{}}
    />
  );
}

registerRoot(RemotionRoot);
