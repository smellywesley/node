import React, { useMemo, useRef } from "react";
import { Canvas, useFrame, useThree } from "@react-three/fiber";
import { Grid, Line, Sparkles } from "@react-three/drei";
import * as THREE from "three";

function ease(t) {
  return t < .5 ? 4 * t * t * t : 1 - ((-2 * t + 2) ** 3) / 2;
}

function segment(progress, chapters) {
  const max = chapters.length - 1;
  const scaled = Math.min(max, Math.max(0, progress * max));
  const index = Math.min(max - 1, Math.floor(scaled));
  return { index, next: Math.min(max, index + 1), local: scaled - index };
}

function lerpArray(a, b, t) {
  return a.map((value, index) => THREE.MathUtils.lerp(value, b[index], t));
}

function CameraRig({ chapters, progress, reduce }) {
  const { camera, pointer } = useThree();
  const look = useMemo(() => new THREE.Vector3(), []);

  useFrame(() => {
    const part = segment(progress, chapters);
    const from = chapters[part.index];
    const to = chapters[part.next];
    const t = reduce ? 0 : ease(part.local);
    const cam = reduce ? chapters[0].camera : lerpArray(from.camera, to.camera, t);
    const target = reduce ? chapters[0].target : lerpArray(from.target, to.target, t);
    const driftX = reduce ? 0 : pointer.x * .28;
    const driftY = reduce ? 0 : pointer.y * .18;

    camera.position.lerp(new THREE.Vector3(cam[0] + driftX, cam[1] + driftY, cam[2]), .08);
    look.lerp(new THREE.Vector3(target[0] + driftX * .35, target[1], target[2]), .09);
    camera.lookAt(look);
  });

  return null;
}

function makeLabelTexture(text, options = {}) {
  const canvas = document.createElement("canvas");
  canvas.width = options.width || 480;
  canvas.height = options.height || 160;
  const ctx = canvas.getContext("2d");
  ctx.clearRect(0, 0, canvas.width, canvas.height);
  ctx.fillStyle = options.fill || "rgba(5, 15, 12, 0.72)";
  ctx.strokeStyle = options.stroke || "rgba(142, 255, 211, 0.28)";
  ctx.lineWidth = 2;
  ctx.beginPath();
  ctx.roundRect(20, 26, canvas.width - 40, canvas.height - 52, options.radius || 28);
  ctx.fill();
  if (options.box !== false) ctx.stroke();
  ctx.font = `${options.weight || 800} ${options.size || 42}px ${options.family || "Inter, Segoe UI, sans-serif"}`;
  ctx.textAlign = "center";
  ctx.textBaseline = "middle";
  ctx.fillStyle = options.color || "#eafff7";
  ctx.shadowColor = options.glow || "rgba(95, 255, 190, 0.72)";
  ctx.shadowBlur = options.shadow ?? 18;
  ctx.fillText(text, canvas.width / 2, canvas.height / 2 + 1);
  const texture = new THREE.CanvasTexture(canvas);
  texture.colorSpace = THREE.SRGBColorSpace;
  texture.needsUpdate = true;
  return texture;
}

function Label({ text, position, scale, color = "#f7fffb", box = true }) {
  const texture = useMemo(() => makeLabelTexture(text, { color, box }), [text, color, box]);
  return (
    <sprite position={position} scale={scale} renderOrder={10}>
      <spriteMaterial map={texture} transparent depthWrite={false} depthTest={false} />
    </sprite>
  );
}

function MathField({ reduce }) {
  const group = useRef(null);
  const glyphs = useMemo(() => [
    ["24", [-7.8, 2.7, -2.2], 1.1],
    ["-15+6", [-6.2, 1.4, -3.4], .8],
    ["12", [5.9, 2.8, -3.6], .72],
    ["5x9", [7.4, 1.1, -2.8], .92],
    ["sqrt", [5.3, .2, -4.4], .72],
    ["{ }", [-8, .1, -4.8], .74],
    ["01", [-2.2, 3.8, -5.4], .48],
    ["run", [2.2, 3.1, -5.8], .42]
  ], []);

  useFrame((state) => {
    if (!group.current || reduce) return;
    group.current.position.y = Math.sin(state.clock.elapsedTime * .35) * .12;
    group.current.rotation.z = Math.sin(state.clock.elapsedTime * .16) * .02;
  });

  return (
    <group ref={group}>
      {glyphs.map(([text, position, size]) => (
        <Label key={`${text}-${position.join(",")}`} text={text} position={position} scale={[size * 1.25, size * .45, 1]} color="rgba(235,255,247,.9)" box={false} />
      ))}
    </group>
  );
}

function WireBox({ position = [0, 0, 0], scale = 1, color = "#dffff5", opacity = .42 }) {
  const points = useMemo(() => {
    const s = scale;
    const v = [
      [-s, -s, -s], [s, -s, -s], [s, s, -s], [-s, s, -s],
      [-s, -s, s], [s, -s, s], [s, s, s], [-s, s, s]
    ];
    return [
      [v[0], v[1], v[2], v[3], v[0]],
      [v[4], v[5], v[6], v[7], v[4]],
      [v[0], v[4]],
      [v[1], v[5]],
      [v[2], v[6]],
      [v[3], v[7]]
    ];
  }, [scale]);

  return (
    <group position={position} rotation={[.44, .62, .08]}>
      {points.map((line, index) => (
        <Line key={index} points={line} color={color} lineWidth={1.2} transparent opacity={opacity} />
      ))}
    </group>
  );
}

function SoftCore({ activeScene, reduce }) {
  const group = useRef(null);
  const active = activeScene === "hero" || activeScene === "console" || activeScene === "ribbon";

  useFrame((state) => {
    if (!group.current || reduce) return;
    const pulse = 1 + Math.sin(state.clock.elapsedTime * 1.4) * .03;
    group.current.scale.lerp(new THREE.Vector3(pulse, pulse, pulse), .08);
    group.current.rotation.y += .002;
  });

  return (
    <group ref={group} position={[0, .35, 0]}>
      <mesh>
        <icosahedronGeometry args={[1.08, 4]} />
        <meshPhysicalMaterial color="#061411" emissive="#4bffb5" emissiveIntensity={active ? .58 : .2} roughness={.18} metalness={.35} clearcoat={.8} transparent opacity={.58} />
      </mesh>
      <mesh scale={[1.5, 1.5, 1.5]}>
        <icosahedronGeometry args={[1.08, 2]} />
        <meshBasicMaterial color="#5dffbd" wireframe transparent opacity={active ? .28 : .12} />
      </mesh>
      <pointLight color="#53ffc0" intensity={active ? 6.8 : 2.8} distance={16} />
      <Label text="NODE" position={[0, 1.78, 0]} scale={[1.7, .54, 1]} color="#f5fffb" />
    </group>
  );
}

function CurvedPath({ points, active, color = "#55ffbb", width = 1.4 }) {
  return (
    <>
      <Line points={points} color="#ffffff" lineWidth={active ? width + 2.2 : width + .8} transparent opacity={active ? .38 : .1} />
      <Line points={points} color={color} lineWidth={active ? width : width * .62} transparent opacity={active ? .82 : .28} />
    </>
  );
}

function RibbonField({ activeScene, reduce }) {
  const group = useRef(null);
  const paths = useMemo(() => {
    const p1 = new THREE.CatmullRomCurve3([
      new THREE.Vector3(-8.5, -1.2, -2.8),
      new THREE.Vector3(-4.4, .4, -1.4),
      new THREE.Vector3(-1.6, .25, -.8),
      new THREE.Vector3(1.8, 1.05, -.2),
      new THREE.Vector3(7.8, 2.4, -1.8)
    ]).getPoints(90);
    const p2 = new THREE.CatmullRomCurve3([
      new THREE.Vector3(-7.8, 2.6, -4.2),
      new THREE.Vector3(-2.8, 2.2, -2.8),
      new THREE.Vector3(1.6, .8, -1.8),
      new THREE.Vector3(7.6, .7, -2.4)
    ]).getPoints(90);
    const p3 = new THREE.CatmullRomCurve3([
      new THREE.Vector3(-5.6, -2.2, -1.2),
      new THREE.Vector3(-3.2, -.4, .1),
      new THREE.Vector3(.4, -.15, .6),
      new THREE.Vector3(4.4, 1.8, .1),
      new THREE.Vector3(8.2, 2.7, -.8)
    ]).getPoints(90);
    return [p1, p2, p3];
  }, []);

  useFrame((state) => {
    if (!group.current || reduce) return;
    group.current.position.x = Math.sin(state.clock.elapsedTime * .18) * .15;
  });

  const active = activeScene === "ribbon" || activeScene === "hero";
  return (
    <group ref={group}>
      {paths.map((points, index) => (
        <CurvedPath key={index} points={points} active={active || index === 0} color={index === 1 ? "#3cd79f" : "#59ffc0"} width={index === 0 ? 2.4 : 1.3} />
      ))}
      <Packet points={paths[0]} reduce={reduce} />
    </group>
  );
}

function Packet({ points, reduce }) {
  const packet = useRef(null);
  useFrame((state) => {
    if (!packet.current || reduce) return;
    const t = (state.clock.elapsedTime * .12) % 1;
    const index = Math.min(points.length - 1, Math.floor(t * points.length));
    packet.current.position.copy(points[index]);
  });

  return (
    <group ref={packet}>
      <mesh>
        <sphereGeometry args={[.08, 18, 12]} />
        <meshBasicMaterial color="#ffffff" />
      </mesh>
      <pointLight color="#75ffd0" intensity={4.2} distance={7} />
    </group>
  );
}

const runNodes = [
  { id: "request", label: "Request", position: [-4.8, .25, -1.2], color: "#f7fffb" },
  { id: "policy", label: "Policy Gate", position: [-2.65, 1.05, -.2], color: "#58ffb9" },
  { id: "control", label: "NODE Control", position: [0, .58, .1], color: "#58ffb9" },
  { id: "sandbox", label: "Sandbox", position: [2.55, .95, -.45], color: "#dffaf2" },
  { id: "cost", label: "Cost Meter", position: [4.15, .12, -1.25], color: "#bfffe9" },
  { id: "audit", label: "Audit Bundle", position: [5.45, 1.12, -.05], color: "#ffffff" }
];

function getRunNodePath() {
  return new THREE.CatmullRomCurve3(runNodes.map((node) => new THREE.Vector3(...node.position))).getPoints(140);
}

function RunNode({ node, index, active, reduce }) {
  const group = useRef(null);
  const color = new THREE.Color(node.color);

  useFrame((state) => {
    if (!group.current || reduce) return;
    const pulse = active ? 1 + Math.sin(state.clock.elapsedTime * 2.2 + index) * .06 : .82;
    group.current.scale.lerp(new THREE.Vector3(pulse, pulse, pulse), .1);
  });

  return (
    <group ref={group} position={node.position}>
      <mesh>
        <icosahedronGeometry args={[index === 2 ? .34 : .23, 2]} />
        <meshPhysicalMaterial color="#061411" emissive={color} emissiveIntensity={active ? 1.4 : .38} roughness={.2} metalness={.38} clearcoat={.8} />
      </mesh>
      <mesh scale={[1.85, 1.85, 1.85]}>
        <sphereGeometry args={[index === 2 ? .34 : .23, 24, 12]} />
        <meshBasicMaterial color={node.color} transparent opacity={active ? .12 : .04} />
      </mesh>
      <pointLight color={node.color} intensity={active ? 2.6 : .5} distance={active ? 6 : 3} />
      <Label text={node.label} position={[0, .52, 0]} scale={[index === 2 ? 1.35 : 1.05, .34, 1]} color={node.color} />
    </group>
  );
}

function HandoffPacket({ points, active, reduce }) {
  const packet = useRef(null);

  useFrame((state) => {
    if (!packet.current || reduce) return;
    const speed = active ? .16 : .06;
    const t = (state.clock.elapsedTime * speed) % 1;
    const index = Math.min(points.length - 1, Math.floor(t * points.length));
    packet.current.position.copy(points[index]);
    packet.current.rotation.y += .03;
  });

  return (
    <group ref={packet}>
      <mesh>
        <tetrahedronGeometry args={[.14, 1]} />
        <meshBasicMaterial color="#ffffff" transparent opacity={active ? .96 : .38} />
      </mesh>
      <pointLight color="#ffffff" intensity={active ? 5.8 : 1.2} distance={6} />
    </group>
  );
}

function HandoffGraph({ activeScene, reduce }) {
  const group = useRef(null);
  const path = useMemo(() => getRunNodePath(), []);
  const active = activeScene === "ribbon";

  useFrame((state) => {
    if (!group.current || reduce) return;
    group.current.position.y = Math.sin(state.clock.elapsedTime * .42) * .08;
    group.current.rotation.y = Math.sin(state.clock.elapsedTime * .14) * .035;
  });

  return (
    <group ref={group} position={[0, -.2, -.25]} scale={active ? 1 : .76}>
      <Line points={path} color="#ffffff" lineWidth={active ? 4 : 1.4} transparent opacity={active ? .28 : .08} />
      <Line points={path} color="#58ffb9" lineWidth={active ? 2.25 : .82} transparent opacity={active ? .78 : .2} />
      <HandoffPacket points={path} active={active} reduce={reduce} />
      {runNodes.map((node, index) => (
        <RunNode key={node.id} node={node} index={index} active={active || index === 2} reduce={reduce} />
      ))}
    </group>
  );
}

function ConsolePlate({ activeScene }) {
  const active = activeScene === "console" || activeScene === "program";
  return (
    <group position={[0, -.32, -.5]} rotation={[-.98, 0, 0]} scale={active ? 1 : .82}>
      <mesh>
        <planeGeometry args={[7.2, 4.1, 1, 1]} />
        <meshPhysicalMaterial color="#07120f" roughness={.2} metalness={.2} transparent opacity={active ? .48 : .24} clearcoat={.7} />
      </mesh>
      <Line points={[[-3.1, -1.6, .02], [-1.4, -.3, .02], [.6, -.2, .02], [1.8, 1.2, .02], [3.2, 1.5, .02]]} color="#eafff7" lineWidth={active ? 2.6 : 1.2} transparent opacity={active ? .58 : .18} />
      <Line points={[[-3.3, .7, .02], [-1.1, .7, .02], [-.2, 1.5, .02], [1.6, 1.5, .02], [3.4, -.2, .02]]} color="#52ffc0" lineWidth={active ? 2 : 1} transparent opacity={active ? .65 : .2} />
      {[-2.4, -.8, 1.1, 2.7].map((x, index) => (
        <mesh key={index} position={[x, index % 2 ? .7 : -.9, .05]}>
          <sphereGeometry args={[.09, 16, 10]} />
          <meshBasicMaterial color={index === 1 ? "#ffffff" : "#57ffc2"} transparent opacity={active ? .9 : .35} />
        </mesh>
      ))}
    </group>
  );
}

function ProgramGhosts({ activeScene }) {
  const active = activeScene === "program";
  return (
    <group position={[1.5, .4, -1.4]} rotation={[.04, -.2, 0]}>
      {[0, 1, 2].map((item) => (
        <group key={item} position={[0, (item - 1) * -.78, item * -.36]} scale={item === 1 ? 1.08 : .86}>
          <mesh>
            <boxGeometry args={[3.8, .52, .04]} />
            <meshPhysicalMaterial color="#07110f" emissive="#47f7b0" emissiveIntensity={active && item === 1 ? .34 : .08} roughness={.2} metalness={.22} transparent opacity={active ? .44 : .16} clearcoat={.7} />
          </mesh>
          <Line points={[[-1.72, 0, .04], [1.72, 0, .04]]} color={item === 1 ? "#f6fffb" : "#4bffb5"} lineWidth={item === 1 ? 1.8 : .9} transparent opacity={active ? .44 : .12} />
        </group>
      ))}
    </group>
  );
}

function ProofChart({ activeScene }) {
  const active = activeScene === "chart";
  const years = ["2024", "2026", "2028"];
  const heights = [.62, 1.3, 2.3];
  return (
    <group position={[3.1, -.72, -1.8]} rotation={[-.72, 0, 0]}>
      <Grid args={[5.8, 3.7]} cellSize={.56} cellThickness={.34} sectionSize={1.68} sectionThickness={.75} cellColor="#285445" sectionColor="#69ffc8" fadeDistance={9} fadeStrength={1.4} />
      {heights.map((height, index) => (
        <group key={years[index]} position={[-2 + index * 2, 0, 0]}>
          <Line points={[[0, 0, 0], [0, height, 0]]} color="#ffffff" lineWidth={active ? 2.8 : 1.2} transparent opacity={active ? .75 : .22} />
          <mesh position={[0, height, 0]}>
            <sphereGeometry args={[.11, 18, 10]} />
            <meshBasicMaterial color={index === 2 ? "#ffffff" : "#66ffc6"} transparent opacity={active ? .95 : .4} />
          </mesh>
          <pointLight position={[0, height, 0]} color="#5cffc1" intensity={active ? 2.6 : .7} distance={4.2} />
          <Label text={years[index]} position={[0, -.48, .3]} scale={[.68, .24, 1]} color="#f7fffb" box={false} />
        </group>
      ))}
    </group>
  );
}

function SceneWorld({ chapters, progress, activeIndex, reduce }) {
  const activeScene = chapters[activeIndex]?.scene || "hero";

  return (
    <>
      <color attach="background" args={["#030605"]} />
      <fog attach="fog" args={["#030605", 10, 32]} />
      <CameraRig chapters={chapters} progress={progress} reduce={reduce} />
      <ambientLight intensity={.52} color="#bffff0" />
      <pointLight position={[-3.5, 5, 7]} intensity={activeScene === "hero" ? 7.6 : 4.2} color="#55ffc0" distance={28} />
      <pointLight position={[6, 4.2, -5]} intensity={3.2} color="#dffff6" distance={22} />
      <spotLight position={[0, 8, 8]} intensity={4.2} angle={.45} penumbra={.8} color="#bffff0" />
      <Sparkles count={reduce ? 16 : 72} scale={[15, 6, 9]} size={1.25} speed={reduce ? 0 : .24} color="#f2fffb" opacity={.36} />
      <MathField reduce={reduce} />
      <RibbonField activeScene={activeScene} reduce={reduce} />
      <HandoffGraph activeScene={activeScene} reduce={reduce} />
      <WireBox position={[-2.7, .15, -1.2]} scale={1.24} opacity={activeScene === "wire" ? .66 : .22} />
      <ConsolePlate activeScene={activeScene} />
      <ProgramGhosts activeScene={activeScene} />
      <ProofChart activeScene={activeScene} />
      <SoftCore activeScene={activeScene} reduce={reduce} />
    </>
  );
}

export default function ReferenceScene({ chapters, progress, activeIndex, reduce }) {
  const visualTest = new URLSearchParams(window.location.search).has("visual-test");
  return (
    <div className="reference-scene" aria-hidden="true">
      <Canvas
        dpr={[1, 1.7]}
        camera={{ position: chapters[0].camera, fov: 44, near: .1, far: 80 }}
        gl={{ antialias: true, alpha: false, powerPreference: "high-performance", preserveDrawingBuffer: visualTest }}
      >
        <SceneWorld chapters={chapters} progress={progress} activeIndex={activeIndex} reduce={reduce} />
      </Canvas>
    </div>
  );
}
