using System;
using System.Drawing;
using System.Linq;
using WinApi.User32;
using Direct3D11 = SharpDX.Direct3D11;
using DXGI = SharpDX.DXGI;

namespace Capture
{
    public class DesktopDuplicationCapture
    {
        public event EventHandler<CaptureFrame> FrameArrived;
        
        private readonly Direct3D11.Device _device;
        private readonly DXGI.OutputDuplication _duplication;
        private readonly int _acquireFrameTimeoutMilliseconds;
        private Direct3D11.Texture2D _texture;
        private Size _textureSize;
        
        public DesktopDuplicationCapture(int acquireFrameTimeoutMilliseconds)
        {
            var output = GetPrimaryMonitorOutput();
            var adapter = output.GetParent<DXGI.Adapter>();
            _device = new Direct3D11.Device(adapter);
            _duplication = output.DuplicateOutput(_device);
            _acquireFrameTimeoutMilliseconds = acquireFrameTimeoutMilliseconds;
        }

        public void Capture(Rectangle rectangle)
        {
            if (NeedsCreatingTexture(rectangle.Size))
            {
                _texture?.Dispose();
                _texture = new Direct3D11.Texture2D(_device, new Direct3D11.Texture2DDescription
                {
                    Width = rectangle.Width,
                    Height = rectangle.Height,
                    MipLevels = 1,
                    ArraySize = 1,
                    Format = DXGI.Format.B8G8R8A8_UNorm,
                    SampleDescription = { Count = 1, Quality = 0 },
                    Usage = Direct3D11.ResourceUsage.Staging,
                    BindFlags = Direct3D11.BindFlags.None,
                    CpuAccessFlags = Direct3D11.CpuAccessFlags.Read,
                    OptionFlags = Direct3D11.ResourceOptionFlags.None,
                });
                _textureSize = rectangle.Size;
            }

            var result = _duplication.TryAcquireNextFrame(_acquireFrameTimeoutMilliseconds, out var frameInfo, out var desktopResource);
            var isTimeout = result == DXGI.ResultCode.WaitTimeout;
            if (isTimeout) return;

            try
            {
                if (frameInfo.LastPresentTime == 0) return;
                using (var desktopTexture = desktopResource.QueryInterface<Direct3D11.Texture2D>())
                {
                    _device.ImmediateContext.CopySubresourceRegion(desktopTexture, 0, new Direct3D11.ResourceRegion(rectangle.Left, rectangle.Top, 0, rectangle.Right, rectangle.Bottom, 1), _texture, 0);
                }

                var dataBox = _device.ImmediateContext.MapSubresource(_texture, 0, Direct3D11.MapMode.Read, Direct3D11.MapFlags.None);
                try
                {
                    var frame = new CaptureFrame(rectangle.Size, dataBox.DataPointer, dataBox.RowPitch);
                    FrameArrived?.Invoke(this, frame);
                }
                finally
                {
                    _device.ImmediateContext.UnmapSubresource(_texture, 0);
                }
            }
            finally
            {
                desktopResource?.Dispose();
                _duplication.ReleaseFrame();
            }
        }

        private bool NeedsCreatingTexture(Size newSize) => _texture == null || _textureSize != newSize;

        private static DXGI.Output1 GetPrimaryMonitorOutput() =>
            new DXGI.Factory1().Adapters1
                .SelectMany(adapter => adapter.Outputs)
                .Where(output =>
                {
                    User32Helpers.GetMonitorInfo(output.Description.MonitorHandle, out var monitorInfo);
                    return monitorInfo.Flags.HasFlag(MonitorInfoFlag.MONITORINFOF_PRIMARY);
                })
                .First()
                .QueryInterface<DXGI.Output1>();
    }
}