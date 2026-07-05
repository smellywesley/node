import React, { useMemo, useRef } from "react";
import { Canvas, useFrame, useThree } from "@react-three/fiber";
import { Float, Grid, Line, RoundedBox, Sparkles } from "@react-three/drei";
import * as THREE from "three";

const nodeSpecs = [
  { id: "intent", label: "Request", position: [-5.4, .42, -1.6], color: "#82f7ff" },
  { id: "policy", label: "Policy Gate", position: [-2.35, .42, 1.65], color: "#48f6b2" },
  { id: "runtime", label: "Sandbox", position: [2.45, .42, 1.5], color: "#9d8cff" },
  { id: "spend", label: "Cost Meter", position: [-1.15, .42, -4.1], color: "#ffe58a" },
  { id: "proof", label: "Audit Bundle", position: [5.05, .42, -2.35], color: "#48f6b2" }
];

const connections = [
  ["intent", "policy"],
  ["policy", "runtime"],
  ["runtime", "proof"],
  ["spend", "runtime"],
  ["intent", "core"],
  ["policy", "core"],
  ["runtime", "core"],
  ["proof", "core"],
  ["spend", "core"]
];

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

function CoreChip({ active }) {
  return (
    <group position={[0, .58, 0]}>
      <RoundedBox args={[2.5, .46, 1.52]} radius={.18} smoothness={8}>
        <meshPhysicalMaterial
          color="#071c17"
          emissive="#48f6b2"
          emissiveIntensity={active ? .72 : .38}
          roughness={.18}
          metalness={.42}
          transmission={.32}
          thickness={.8}
          transparent
          opacity={.92}
        />
      </RoundedBox>
      <LabelSprite text="NODE CONTROL" color="#73ffc9" position={[0, .72, 0]} scale={[2.15, .6, 1]} />
    </group>
  );
}

function NodeOrb({ spec, active, reduce }) {
  const group = useRef(null);
  const color = new THREE.Color(spec.color);

  useFrame((state) => {
    if (!group.current || reduce) return;
    const pulse = Math.sin(state.clock.elapsedTime * 2.1 + spec.position[0]) * .04;
    const scale = active ? 1.24 + pulse : .92;
    group.current.scale.lerp(new THREE.Vector3(scale, scale, scale), .08);
  });

  const content = (
    <group ref={group} position={spec.position}>
      <mesh>
        <sphereGeometry args={[.34, 36, 18]} />
        <meshPhysicalMaterial
          color="#071713"
          emissive={color}
          emissiveIntensity={active ? 1.05 : .32}
          roughness={.24}
          metalness={.42}
          clearcoat={.6}
        />
      </mesh>
      <mesh rotation={[Math.PI / 2, 0, 0]}>
        <torusGeometry args={[.62, .016, 12, 86]} />
        <meshBasicMaterial color={color} transparent opacity={active ? .9 : .42} />
      </mesh>
      <LabelSprite text={spec.label} color={spec.color} position={[0, .92, 0]} scale={[1.45, .52, 1]} />
    </group>
  );

  if (reduce) return content;
  return <Float speed={1.25} rotationIntensity={.08} floatIntensity={.12}>{content}</Float>;
}

function NetworkLines({ activeId }) {
  const positions = useMemo(() => {
    const map = new Map(nodeSpecs.map((node) => [node.id, node.position]));
    map.set("core", [0, .58, 0]);
    return map;
  }, []);

  return connections.map(([from, to]) => {
    const active = activeId === "core" || activeId === from || activeId === to;
    return (
      <Line
        key={`${from}-${to}`}
        points={[positions.get(from), positions.get(to)]}
        color={active ? "#48f6b2" : "#1f6b54"}
        lineWidth={active ? 2.4 : 1.1}
        transparent
        opacity={active ? .82 : .35}
      />
    );
  });
}

function MovingPacket({ reduce }) {
  const packet = useRef(null);
  const points = useMemo(() => [
    new THREE.Vector3(-5.4, .68, -1.6),
    new THREE.Vector3(-2.35, .72, 1.65),
    new THREE.Vector3(2.45, .72, 1.5),
    new THREE.Vector3(-1.15, .72, -4.1),
    new THREE.Vector3(5.05, .72, -2.35)
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
        <sphereGeometry args={[.14, 24, 12]} />
        <meshBasicMaterial color="#7df7ff" />
      </mesh>
      <pointLight intensity={2.4} distance={5} color="#7df7ff" />
    </group>
  );
}

function BlockedActionMarker({ activeId }) {
  const show = activeId === "policy" || activeId === "core";
  return (
    <group position={[-2.36, 1.18, 2.72]} scale={show ? 1 : .72}>
      <mesh>
        <boxGeometry args={[.72, .05, .72]} />
        <meshBasicMaterial color="#ff6f7d" transparent opacity={show ? .64 : .24} />
      </mesh>
      <LabelSprite text="frontend/* blocked" color="#ff8b96" position={[0, .42, 0]} scale={[1.9, .56, 1]} />
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
        <NetworkLines activeId={activeId} />
        <MovingPacket reduce={reduce} />
        <BlockedActionMarker activeId={activeId} />
        <CoreChip active={activeId === "core"} />
        {nodeSpecs.map((spec) => (
          <NodeOrb key={spec.id} spec={spec} active={activeId === "core" || activeId === spec.id} reduce={reduce} />
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
