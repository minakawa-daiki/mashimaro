using System;
using System.Drawing;

namespace Capture
{
    public readonly struct CaptureFrame
    {
        public Size Size { get;  }
        public IntPtr Buffer { get; }
        public int RowPitch { get; }

        internal CaptureFrame(Size size, IntPtr buffer, int rowPitch) => (Size, Buffer, RowPitch) = (size, buffer, rowPitch);
    }
}