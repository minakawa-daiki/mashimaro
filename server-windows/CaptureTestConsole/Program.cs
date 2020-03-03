using System;
using System.Diagnostics;
using System.IO;
using System.Linq;
using System.Net.Sockets;
using System.Threading;
using Windows.Graphics.Capture;
using Windows.Graphics.DirectX;
using Helpers;
using SharpDX;
using SharpDX.Direct3D11;

namespace CaptureTestConsole
{
    class Program
    {
        static void Main(string[] args)
        {
            var process = Process.GetProcessesByName("Sakura").First();
            var handle = process.MainWindowHandle;
            var captureItem = CaptureHelper.CreateItemForWindow(handle);
            captureItem.Closed += (sender, o) => { Console.WriteLine($"capture item closed"); };

            var videoTx = NativeMethods.RtpVideoTx_new(-1, VideoFormat.BGRA_8bit);
            if (NativeMethods.RtpVideoTx_addDestination(videoTx, "192.168.10.101", 9999) != 0)
            {
                throw new Exception("failed to videoTx addDestination");
            }

            if (NativeMethods.RtpVideoTx_setSSRC(videoTx, 0) != 0)
            {
                throw new Exception("failed to videoTx setSSRC");
            }

            using var device = Direct3D11Helper.CreateDevice();
            using var d3dDevice = Direct3D11Helper.CreateSharpDXDevice(device);
            var stage = Direct3D11Helper.CreateSharpDXStagingTexture2D(d3dDevice, captureItem.Size.Width, captureItem.Size.Height);
            
            using var framePool = Direct3D11CaptureFramePool.CreateFreeThreaded(
                device,
                DirectXPixelFormat.B8G8R8A8UIntNormalized,
                2,
                captureItem.Size);
            using var session = framePool.CreateCaptureSession(captureItem);

            var lastSize = captureItem.Size;
            uint timestamp = 0;
            const uint timestampStep = 90000 / 120;
            framePool.FrameArrived += (sender, o) =>
            {
                var newSize = false;
                // Thread.Sleep(16);
                using var frame = sender.TryGetNextFrame();
                if (frame.ContentSize.Width != lastSize.Width || frame.ContentSize.Height != lastSize.Height)
                {
                    newSize = true;
                    lastSize = frame.ContentSize;
                }

                using var tex = Direct3D11Helper.CreateSharpDXTexture2D(frame.Surface);
                d3dDevice.ImmediateContext.CopyResource(tex, stage);
                
                DataStream ds;
                var dataBox = d3dDevice.ImmediateContext.MapSubresource(stage, 0, 0, MapMode.Read, MapFlags.None, out ds);
                try
                {
                    var width = frame.ContentSize.Width;
                    var height = frame.ContentSize.Height;
                    if (NativeMethods.RtpVideoTx_beginFrame(videoTx, timestamp) != 0)
                    {
                        throw new Exception("failed to videoTx beginFrame");
                    }
                    timestamp += timestampStep;
                    for (var row = 0; row < height; row++)
                    {
                        var lineBytes = width * 4; /* 4bytes per pixel */
                        var isLastLine = row == height - 1;
                        var flags = isLastLine ? 0x01 : 0x00;
                        if (NativeMethods.RtpVideoTx_addLine(videoTx, (uint) row, 0, (uint) lineBytes,
                            ds.PositionPointer, (uint) flags) != 0)
                        {
                            throw new Exception("failed to videoTx addLine");
                        }
                        ds.Position += dataBox.RowPitch;
                    }
                    // Console.WriteLine($"write frame {width}x{height}");
                }
                finally
                {
                    d3dDevice.ImmediateContext.UnmapSubresource(stage, 0);
                    ds.Dispose();
                }

                if (newSize)
                {
                    stage = Direct3D11Helper.CreateSharpDXStagingTexture2D(d3dDevice, lastSize.Width, lastSize.Height);
                    framePool.Recreate(
                        device, 
                        DirectXPixelFormat.B8G8R8A8UIntNormalized,
                        2,
                        lastSize);
                }
            };
            session.StartCapture();
            while (true)
            {
                Thread.Sleep(20);
            }
        }
    }
}