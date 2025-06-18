import React, { useEffect, useState, useRef } from "react";
import "./App.css";

function App() {
  const [data, setData] = useState([]);
  const canvasRef = useRef(null);
  const animationRef = useRef(null);
  const startTimeRef = useRef(null);
  const positionsRef = useRef([]);

  useEffect(() => {
    fetch("http://localhost:8080/simulate")
      .then((res) => res.json())
      .then((json) => {
        setData(json);
        const points = buildPathPoints(json);
        positionsRef.current = points;
        drawStaticTrack(points);
        startAnimation();
      })
      .catch((err) => console.error("Failed to fetch:", err));

    return () => cancelAnimationFrame(animationRef.current);
  }, []);

  const buildPathPoints = (data) => {
    let x = 300, y = 300, angle = 0;
    const points = [{ x, y, angle }];
    const scale = .5;

    for (let i = 1; i < data.length; i++) {
      const segment = data[i];
      const prev = points[points.length - 1];

      let dx = 0;
      let dy = 0;

      if (segment.SegmentType === "straight") {
        dx = scale * (segment.Distance - data[i - 1].Distance) * Math.cos(angle);
        dy = scale * (segment.Distance - data[i - 1].Distance) * Math.sin(angle);
      } else if (segment.SegmentType === "turn") {
        // Fake left/right turns using angle change
        angle += Math.PI / 4; // 45 degree turn left for simplicity
        dx = scale * 20 * Math.cos(angle);
        dy = scale * 20 * Math.sin(angle);
      }

      x = prev.x + dx;
      y = prev.y + dy;
      points.push({ x, y, angle });
    }

    return points;
  };
    const drawStaticTrack = (points) => {
    const ctx = canvasRef.current.getContext("2d");
    ctx.clearRect(0, 0, 800, 600);

    ctx.beginPath();
    ctx.moveTo(points[0].x, points[0].y);
    for (const pt of points) {
      ctx.lineTo(pt.x, pt.y);
    }
    ctx.strokeStyle = "#333";
    ctx.lineWidth = 2;
    ctx.stroke();
  };

  const startAnimation = () => {
    startTimeRef.current = performance.now();
    animationRef.current = requestAnimationFrame(animateCar);
  };

const animateCar = (timestamp) => {
  if (!data.length || !positionsRef.current.length) {
    return; // nothing to animate yet
  }

  const lastTime = data[data.length - 1]?.Time;
  const elapsedSec = (timestamp - startTimeRef.current) / 1000;

  // Stop animation after last time point
  if (elapsedSec >= lastTime) {
    drawStaticTrack(positionsRef.current);

    // Draw car at final point
    const lastPoint = positionsRef.current[positionsRef.current.length - 1];
    const ctx = canvasRef.current.getContext("2d");
    ctx.beginPath();
    ctx.arc(lastPoint.x, lastPoint.y, 6, 0, 2 * Math.PI);
    ctx.fillStyle = "red";
    ctx.fill();
    ctx.fillStyle = "black";
    ctx.fillText(`Segment ${data.length}`, 10, 20);

    return; // stop animating
  }

  let i = 0;
  while (i < data.length - 1 && data[i + 1]?.Time < elapsedSec) {
    i++;
  }

  const p1 = positionsRef.current[i];
  const p2 = positionsRef.current[i + 1] || p1;
  const t1 = data[i]?.Time || 0;
  const t2 = data[i + 1]?.Time || t1;

  const ratio = t2 !== t1 ? (elapsedSec - t1) / (t2 - t1) : 0;

  const x = p1.x * (1 - ratio) + p2.x * ratio;
  const y = p1.y * (1 - ratio) + p2.y * ratio;

  const ctx = canvasRef.current.getContext("2d");
  drawStaticTrack(positionsRef.current);

  ctx.beginPath();
  ctx.arc(x, y, 6, 0, 2 * Math.PI);
  ctx.fillStyle = "red";
  ctx.fill();

  ctx.fillStyle = "black";
  ctx.font = "16px sans-serif";
  ctx.fillText(`Segment ${i + 1}`, 10, 20);

  animationRef.current = requestAnimationFrame(animateCar);
};

  return (
    <div className="App">
      <h1>Solar Car Track Simulation</h1>

      <canvas
        ref={canvasRef}
        width={800}
        height={600}
        style={{ border: "2px solid black", margin: "20px 0" }}
      />

      <table>
        <thead>
          <tr>
            <th>Segment</th>
            <th>Type</th>
            <th>Distance (m)</th>
            <th>Speed (m/s)</th>
            <th>Energy (Wh)</th>
            <th>Time (s)</th>
          </tr>
        </thead>
        <tbody>
          {data.map((dp, idx) => (
            <tr key={idx}>
              <td>{idx + 1}</td>
              <td>{dp.SegmentType}</td>
              <td>{dp.Distance.toFixed(2)}</td>
              <td>{dp.Speed.toFixed(2)}</td>
              <td>{dp.Energy.toFixed(2)}</td>
              <td>{dp.Time.toFixed(2)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export default App;
