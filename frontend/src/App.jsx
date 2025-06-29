import React, { useEffect, useState, useRef } from "react";
import "./App.css";

function App() {
  const [data, setData] = useState([]);
  const canvasRef = useRef(null);
  const animationRef = useRef(null);
  const startTimeRef = useRef(0);
  const pointsRef = useRef([]);

  // Build a list of { x, y, angle, time } including the first segment
  function buildTimedPoints(data) {
    let x = 150,
      y = 250,
      angle = 0;
    const scale = 2;
    // start at t = 0
    const pts = [{ x, y, angle, time: 0 }];

    let prevDist = 0,
      prevTime = 0;

    for (const seg of data) {
      const d = (seg.Distance - prevDist) * scale;
      const t1 = prevTime,
        t2 = seg.Time;
      prevDist = seg.Distance;
      prevTime = seg.Time;

      if (seg.SegmentType === "turn") {
        const rawR = seg.Radius * scale;
        const dir = rawR >= 0 ? 1 : -1;
        const r = Math.abs(rawR);
        const theta = d / r;

        // subdivide arc: ≤1° per step or ~2px arc length
        const steps = Math.max(
          Math.ceil(Math.abs(theta) / (Math.PI / 180)),
          Math.ceil(d / 2)
        );

        const cx = x - dir * r * Math.sin(angle);
        const cy = y + dir * r * Math.cos(angle);

        for (let s = 1; s <= steps; s++) {
          const frac = s / steps;
          const a = angle + dir * theta * frac;
          const px = cx + dir * r * Math.sin(a);
          const py = cy - dir * r * Math.cos(a);
          const time = t1 + (t2 - t1) * frac;
          pts.push({ x: px, y: py, angle: a, time });
        }

        angle += dir * theta;
        x = pts[pts.length - 1].x;
        y = pts[pts.length - 1].y;
      } else {
        // straight: ~5px per step
        const steps = Math.max(1, Math.ceil(d / 2));
        for (let s = 1; s <= steps; s++) {
          const frac = s / steps;
          const px = x + Math.cos(angle) * d * frac;
          const py = y + Math.sin(angle) * d * frac;
          const time = t1 + (t2 - t1) * frac;
          pts.push({ x: px, y: py, angle, time });
        }
        x += Math.cos(angle) * d;
        y += Math.sin(angle) * d;
      }
    }

    return pts;
  }

  function drawTrack() {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext("2d");
    ctx.clearRect(0, 0, canvas.width, canvas.height);

    const pts = pointsRef.current;
    if (pts.length < 2) return;
    ctx.beginPath();
    ctx.moveTo(pts[0].x, pts[0].y);
    for (const p of pts) ctx.lineTo(p.x, p.y);
    ctx.strokeStyle = "#333";
    ctx.lineWidth = 2;
    ctx.stroke();
  }

  function animate(timestamp) {
    const pts = pointsRef.current;
    if (!pts.length) return;

    const elapsed = (timestamp - startTimeRef.current) / 1000;
    const totalTime = pts[pts.length - 1].time;

    if (elapsed >= totalTime) {
      drawTrack();
      const last = pts[pts.length - 1];
      const ctx = canvasRef.current.getContext("2d");
      ctx.beginPath();
      ctx.arc(last.x, last.y, 6, 0, 2 * Math.PI);
      ctx.fillStyle = "red";
      ctx.fill();
      return;
    }

    let idx = pts.findIndex((p) => p.time > elapsed);
    if (idx <= 0) idx = 0;
    const p = pts[idx];

    drawTrack();
    const ctx = canvasRef.current.getContext("2d");
    ctx.beginPath();
    ctx.arc(p.x, p.y, 8, 0, 2 * Math.PI);
    ctx.fillStyle = "red";
    ctx.fill();
    ctx.font = "16px sans-serif";
    ctx.fillStyle = "black";
    ctx.fillText(`t=${elapsed.toFixed(2)}s`, 10, 20);

    animationRef.current = requestAnimationFrame(animate);
  }

  function runSimulation() {
    fetch("http://localhost:8080/simulate")
      .then((r) => r.json())
      .then((json) => {
        console.log("running");
        setData(json);
        pointsRef.current = buildTimedPoints(json);
        drawTrack();
        startTimeRef.current = performance.now();
        animationRef.current = requestAnimationFrame(animate);
      })
      .catch((err) => console.error(err));
  }

  // handles the textbox submission to the backend

  //values displayed
  const [solarrate, setSolarrate] = useState("");
  const [maxspeed, setMaxspeed] = useState("");
  const [maxgforce, setMaxgforce] = useState("");

  //temp values for button onchange
  const [temp_solarrate, setTemp_solarrate] = useState("");
  const [temp_maxspeed, setTemp_maxspeed] = useState("");
  const [temp_maxgforce, setTemp_maxgforce] = useState("");

  //turns final values into temp values and sends POST request to backend
  function handleClick() {
    const solar = parseFloat(temp_solarrate);
    const speed = parseFloat(temp_maxspeed);
    const gforce = parseFloat(temp_maxgforce);

    // Update state (for display b/c React variables are asynch and fetch is called with old/null data)
    setSolarrate(solar);
    setMaxspeed(speed);
    setMaxgforce(gforce);
    fetch("http://localhost:8080/input", {
      method: "POST",
      headers: {
        "Content-type": "application/json",
      },
      body: JSON.stringify({
        maxSpeed: speed,
        maxGForce: gforce,
        solarRate: solar,
      }),
    })
      .then(() => {
        console.log("Input successfully sent");
      })
      .then((data) => {
        console.log("Response from backend:", data);
      })
      .catch((err) => {
        console.error("Error:", err);
      });
  }
  return (
    <div className="App">
      <h1>Solar Car Track Simulation</h1>
      <canvas
        ref={canvasRef}
        width={800}
        height={600}
        style={{ border: "2px solid black", marginBottom: 20 }}
      />
      <div>
        <table>
          <thead>
            <tr>
              <th>Seg</th>
              <th>Type</th>
              <th>Dist (m)</th>
              <th>Speed</th>
              <th>Energy</th>
              <th>Time</th>
            </tr>
          </thead>
          <tbody>
            {data.map((d, i) => (
              <tr key={i}>
                <td>{i + 1}</td>
                <td>{d.SegmentType}</td>
                <td>{d.Distance.toFixed(2)}</td>
                <td>{d.Speed.toFixed(2)}</td>
                <td>{d.Energy.toFixed(2)}</td>
                <td>{d.Time.toFixed(2)}</td>
              </tr>
            ))}
          </tbody>
        </table>
        <input
          style={{ margin: "50px" }}
          type="text"
          placeholder="Current Solar Panel Yield"
          value={temp_solarrate}
          onChange={(e) => {
            setTemp_solarrate(e.target.value);
          }}
        />
        <input
          style={{ margin: "50px" }}
          type="text"
          placeholder="Target Max Speed"
          value={temp_maxspeed}
          onChange={(e) => {
            setTemp_maxspeed(e.target.value);
          }}
        />
        <input
          style={{ margin: "50px" }}
          type="text"
          placeholder="Max G-Force"
          value={temp_maxgforce}
          onChange={(e) => {
            setTemp_maxgforce(e.target.value);
          }}
        />
        <button
          onClick={() => {
            handleClick();
          }}
        >
          Submit
        </button>
        <button style={{ margin: "30px" }} onClick={runSimulation}>
          Run
        </button>
        <div style={{ display: "flex", flexDirection: "column" }}>
          <h3 style={{ margin: 0, padding: 0 }}>
            Current Solar Yield: {solarrate}
          </h3>
          <h3 style={{ margin: 0, padding: 0 }}>
            Current Max Speed: {maxspeed}
          </h3>
          <h3 style={{ margin: 0, padding: 0 }}>
            Current Max G-force: {maxgforce}
          </h3>
        </div>
      </div>
    </div>
  );
}

export default App;
