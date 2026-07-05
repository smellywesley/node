import React, { useMemo, useRef } from "react";
import { Canvas, useFrame, useThree } from "@react-three/fiber";
import { Float, Grid, Line, Sparkles } from "@react-three/drei";
import * as THREE from "three";

const nodeSpecs = [
  { id: "intent", label: "Request", position: [-5.6, .78, -1.6], color: "#82f7ff" },
  { id: "policy", label: "Policy Gate", position: [-2.75, .78, 2.2], color: "#48f6b2" },
  { id: "runtime", label: "Sandbox", position: [3.05, .78, 1.82], color: "#9d8cff" },
  { id: "spend", label: "Cost Meter", position: [-1.2, .78, -4.45], color: "#ffe58a" },
  { id: "proof", label: "Audit Bundle", position: [5.25, .78, -2.42], color: "#48f6b2" }
];

const connections = [
  ["intent", "policy"],
  ["policy", "core"],
  ["core", "runtime"],
  ["runtime", "spend"],
  ["spend", "proof"],
  ["intent", "core"],
  ["proof", "core"]
];

const corePosition = [0, .9, 0];

function easeInOutPower2(t) {
  return t < .5 ? 2 * t * t : 1 - ((-2 * t + 2) ** 2) / 2;
}

function getSegment(progress, chapters) {
  const max = chapters.length - 1;
  const scaled = Math.min(max, Math.max(0, progress * max));
  const index = Math.min(max - 1, Math.floor(scaled));
  const local = scaled - index;
  return { index, nextIndex: Math.min(max, index + 1), local };
}

function lerpArray(a, b, t) {
  return a.map((value, index) => THREE.MathUtils.lerp(value, b[index], t));
}

function makeLabelTexture(text, color) {
  const canvas = document.createElement("canvas");
  canvas.width = 360;
  canvas.height = 128;
  const ctx = canvas.getContext("2d");
  ctx.clearRect(0, 0, canvas.width, canvas.height);
  ctx.fillStyle = "rgba(3, 13, 11, 0.78)";
  ctx.strokeStyle = "rgba(199, 255, 233, 0.34)";
  ctx.lineWidth = 2;
  ctx.beginPath();
  ctx.roundRect(24, 28, 312, 72, 22);
  ctx.fill();
  ctx.stroke();
  ctx.fillStyle = color;
  ctx.font = "900 34px system-ui, -apple-system, BlinkMacSystemFont, Segoe UI, sans-serif";
  ctx.textAlign = "center";
  ctx.textBaseline = "middle";
  ctx.fillText(text, canvas.width / 2, 64);
  const texture = new THREE.CanvasTexture(canvas);
  texture.colorSpace = THREE.SRGBColorSpace;
  texture.needsUpdate = true;
  return texture;
}

function LabelSprite({ text, color = "#73ffc9", position = [0, 0, 0], scale = [1.6, .56, 1] }) {
  const texture = useMemo(() => makeLabelTexture(text, color), [text, color]);
  return (
    <sprite position={position} scale={scale} renderOrder={10}>
      <spriteMaterial map={texture} transparent depthWrite={false} depthTest={false} />
    </sprite>
  );
}

function SceneCamera({ chapters, progress, reduce }) {
  const { camera, pointer } = useThree();
  const lookAt = useMemo(() => new THREE.Vector3(), []);
  const active = getSegment(progress, chapters);
  const from = chapters[active.index];
  const to = chapters[active.nextIndex];
  const eased = reduce ? 0 : easeInOutPower2(active.local);
  const cameraTarget = reduce ? chapters[0].camera : lerpArray(from.camera, to.camera, eased);
  const lookTarget = reduce ? chapters[0].target : lerpArray(from.target, to.target, eased);

  useFrame(() => {
    const px = reduce ? 0 : pointer.x * .42;
    const py = reduce ? 0 : pointer.y * .24;
    camera.position.lerp(new THREE.Vector3(cameraTarget[0] + px, cameraTarget[1] + py, cameraTarget[2]), .075);
    lookAt.lerp(new THREE.Vector3(lookTarget[0] + px * .45, lookTarget[1], lookTarget[2]), .09);
    camera.lookAt(lookAt);
  });

  return null;
}

function organicOffsets(seed) {
  return Array.from({ length: 5 }, (_, index) => {
    const t = seed + index * 1.47;
    return [
      Math.sin(t) * .32,
      Math.cos(t * 1.2) * .18,
      Math.sin(t * .7) * .3
    ];
  });
}

function BrainCore({ active, reduce }) {
  const group = useRef(null);
  const lobes = useMemo(() => [
    { position: [-.44, .08, .02], scale: [.82, .52, .68] },
    { position: [.42, .06, .03], scale: [.78, .5, .64] },
    { position: [-.04, .28, -.1], scale: [.88, .48, .7] },
    { position: [.02, -.2, .08], scale: [.62, .36, .48] },
    { position: [.08, -.48, .02], scale: [.3, .52, .28] }
  ], []);

  useFrame((state) => {
    if (!group.current || reduce) return;
    const pulse = 1 + Math.sin(state.clock.elapsedTime * 1.6) * .026;
    group.current.scale.lerp(new THREE.Vector3(pulse, pulse, pulse), .08);
  });

  return (
    <group ref={group} position={corePosition}>
      {lobes.map((lobe, index) => (
        <mesh key={index} position={lobe.position} scale={lobe.scale}>
          <sphereGeometry args={[.74, 36, 18]} />
          <meshPhysicalMaterial
            color="#071c17"
            emissive="#48f6b2"
            emissiveIntensity={active ? .72 : .38}
            roughness={.24}
            metalness={.36}
            clearcoat={.7}
            transparent
            opacity={.9}
          />
        </mesh>
      ))}
      {organicOffsets(1.6).map((offset, index) => (
        <Line
          key={`brain-filament-${index}`}
          points={[
            [offset[0] - .36, offset[1], offset[2] - .18],
            [offset[0] * .3, offset[1] + .15, offset[2] * .22],
            [offset[0] + .38, offset[1] - .02, offset[2] + .18]
          ]}
          color="#b7ffe5"
          lineWidth={active ? 1.6 : .8}
          transparent
          opacity={active ? .55 : .28}
        />
      ))}
      <mesh position={[0, .02, 0]}>
        <sphereGeometry args={[1.18, 48, 24]} />
        <meshPhysicalMaterial
          color="#061813"
          emissive="#48f6b2"
          emissiveIntensity={active ? .2 : .1}
          roughness={.1}
          metalness={.12}
          transparent
          opacity={.16}
        />
      </mesh>
      <LabelSprite text="NODE CONTROL" color="#73ffc9" position={[0, 1.22, 0]} scale={[2.15, .6, 1]} />
    </group>
  );
}

function NeuronNode({ spec, active, reduce }) {
  const group = useRef(null);
  const color = new THREE.Color(spec.color);
  const branches = useMemo(() => {
    return organicOffsets(spec.position[0] + spec.position[2]).map((offset, index) => {
      const start = [0, 0, 0];
      const mid = [offset[0] * .9, .1 + offset[1] * .4, offset[2] * .9];
      const end = [offset[0] * 1.85, .12 + offset[1], offset[2] * 1.85];
      return { key: `${spec.id}-${index}`, points: [start, mid, end] };
    });
  }, [spec.id, spec.position]);

  useFrame((state) => {
    if (!group.current || reduce) return;
    const pulse = Math.sin(state.clock.elapsedTime * 2.1 + spec.position[0]) * .04;
    const scale = active ? 1.24 + pulse : .92;
    group.current.scale.lerp(new THREE.Vector3(scale, scale, scale), .08);
  });

  const content = (
    <group ref={group} position={spec.position}>
      {branches.map((branch) => (
        <Line
          key={branch.key}
          points={branch.points}
          color={spec.color}
          lineWidth={active ? 1.7 : .85}
          transparent
          opacity={active ? .78 : .35}
        />
      ))}
      <mesh>
        <icosahedronGeometry args={[.34, 2]} />
        <meshPhysicalMaterial
          color="#071713"
          emissive={color}
          emissiveIntensity={active ? 1.28 : .42}
          roughness={.24}
          metalness={.42}
          clearcoat={.6}
        />
      </mesh>
      {spec.id === "spend" ? (
        <group position={[0, -.5, 0]}>
          {[-.38, -.19, 0, .19, .38].map((x, index) => (
            <Line
              key={`cost-tick-${index}`}
              points={[[x, 0, -.16], [x + .05, .18 + index * .04, .16]]}
              color="#ffe58a"
              lineWidth={active ? 2.1 : 1.1}
              transparent
              opacity={active ? .88 : .42}
            />
          ))}
        </group>
      ) : null}
      <LabelSprite text={spec.label} color={spec.color} position={[0, .92, 0]} scale={[1.45, .52, 1]} />
    </group>
  );

  if (reduce) return content;
  return <Float speed={1.25} rotationIntensity={.08} floatIntensity={.12}>{content}</Float>;
}

function NeuralPathways({ activeId }) {
  const positions = useMemo(() => {
    const map = new Map(nodeSpecs.map((node) => [node.id, node.position]));
    map.set("core", corePosition);
    return map;
  }, []);

  return connections.map(([from, to]) => {
    const active = activeId === "core" || activeId === from || activeId === to;
    const start = positions.get(from);
    const end = positions.get(to);
    const mid = [
      (start[0] + end[0]) / 2,
      Math.max(start[1], end[1]) + (active ? 1.05 : .62),
      (start[2] + end[2]) / 2 + Math.sin(start[0] + end[2]) * .48
    ];
    return (
      <Line
        key={`${from}-${to}`}
        points={[start, mid, end]}
        color={active ? "#48f6b2" : "#1f6b54"}
        lineWidth={active ? 2.7 : 1.1}
        transparent
        opacity={active ? .86 : .32}
      />
    );
  });
}

function MovingPacket({ reduce }) {
  const packet = useRef(null);
  const points = useMemo(() => [
    new THREE.Vector3(-5.6, 1.1, -1.6),
    new THREE.Vector3(-2.75, 1.16, 2.2),
    new THREE.Vector3(...corePosition),
    new THREE.Vector3(3.05, 1.16, 1.82),
    new THREE.Vector3(-1.2, 1.16, -4.45),
    new THREE.Vector3(5.25, 1.16, -2.42)
  ], []);

  useFrame((state) => {
    if (!packet.current || reduce) return;
    const route = (state.clock.elapsedTime * .22) % (points.length - 1);
    const index = Math.floor(route);
    const local = route - index;
    packet.current.position.lerpVectors(points[index], points[index + 1], local);
  });

  return (
    <group ref={packet}>
      <mesh>
        <tetrahedronGeometry args={[.18, 1]} />
        <meshBasicMaterial color="#7df7ff" transparent opacity={.95} />
      </mesh>
      <pointLight intensity={2.4} distance={5} color="#7df7ff" />
    </group>
  );
}

function BlockedActionMarker({ activeId }) {
  const show = activeId === "policy" || activeId === "core";
  return (
    <group position={[-2.7, 1.28, 3.02]} scale={show ? 1 : .72}>
      <Line points={[[-.56, -.18, 0], [.56, .18, 0]]} color="#ff6f7d" lineWidth={show ? 4.2 : 2.2} transparent opacity={show ? .86 : .32} />
      <Line points={[[-.56, .18, 0], [.56, -.18, 0]]} color="#ff6f7d" lineWidth={show ? 4.2 : 2.2} transparent opacity={show ? .86 : .32} />
      <pointLight intensity={show ? 3.8 : 1.2} distance={5} color="#ff6f7d" />
      <LabelSprite text="frontend/* blocked" color="#ff8b96" position={[0, .5, 0]} scale={[1.9, .56, 1]} />
    </group>
  );
}

function ArchitectureScene({ chapters, progress, reduce }) {
  const group = useRef(null);
  const light = useRef(null);
  const activeChapter = chapters[Math.round(Math.min(chapters.length - 1, progress * (chapters.length - 1)))];
  const activeId = activeChapter.focus;

  useFrame((state) => {
    if (group.current && !reduce) {
      group.current.rotation.y = Math.sin(state.clock.elapsedTime * .18) * .045;
    }
    if (light.current) {
      light.current.position.x = -4 + state.pointer.x * 5;
      light.current.position.z = 5 + state.pointer.y * 4;
    }
  });

  return (
    <>
      <color attach="background" args={["#020b09"]} />
      <SceneCamera chapters={chapters} progress={progress} reduce={reduce} />
      <ambientLight intensity={.68} color="#82f7ff" />
      <pointLight ref={light} position={[-4, 6, 5]} intensity={6.2} color="#48f6b2" distance={30} />
      <pointLight position={[6, 5, -7]} intensity={3.2} color="#9d8cff" distance={32} />
      <fog attach="fog" args={["#020403", 14, 34]} />
      <group ref={group}>
        <Grid
          position={[0, -.04, 0]}
          args={[24, 24]}
          cellSize={.8}
          cellThickness={.42}
          sectionSize={4}
          sectionThickness={.95}
          cellColor="#17634d"
          sectionColor="#48f6b2"
          fadeDistance={24}
          fadeStrength={1.45}
          infiniteGrid
        />
        <Sparkles count={reduce ? 16 : 58} scale={[14, 3, 11]} size={reduce ? 1.2 : 1.8} speed={reduce ? 0 : .32} color="#73ffc9" opacity={.45} />
        <NeuralPathways activeId={activeId} />
        <MovingPacket reduce={reduce} />
        <BlockedActionMarker activeId={activeId} />
        <BrainCore active={activeId === "core" || activeId === "policy"} reduce={reduce} />
        {nodeSpecs.map((spec) => (
          <NeuronNode key={spec.id} spec={spec} active={activeId === "core" || activeId === spec.id} reduce={reduce} />
        ))}
      </group>
    </>
  );
}

export default function SceneViewport({ chapters, progress, reduce }) {
  const visualTest = new URLSearchParams(window.location.search).has("visual-test");
  return (
    <div className="scene-viewport" aria-hidden="true">
      <Canvas
        dpr={[1, 1.7]}
        camera={{ position: chapters[0].camera, fov: 43, near: .1, far: 80 }}
        gl={{ antialias: true, alpha: false, powerPreference: "high-performance", preserveDrawingBuffer: visualTest }}
      >
        <ArchitectureScene chapters={chapters} progress={progress} reduce={reduce} />
      </Canvas>
    </div>
  );
}
