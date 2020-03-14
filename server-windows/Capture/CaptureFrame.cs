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

        public byte[] GetBytes()
        {
            const int bytesPerPixel = 4; // R,G,B,A
            var bytes = new byte[Size.Width * Size.Height * bytesPerPixel];
            var srcCursor = Buffer;
            var destCursor = 0;
            for (int row = 0; row < Size.Height; row++)
            {
                var rowBytes = Size.Width * bytesPerPixel;
                unsafe
                {
                    var src = new ReadOnlySpan<byte>(srcCursor.ToPointer(), rowBytes);
                    var dest = new Span<byte>(bytes, destCursor, rowBytes);
                    src.CopyTo(dest);
                }
                srcCursor += RowPitch;
                destCursor += rowBytes;
            }
            return bytes;
        }
    }
}