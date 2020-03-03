using System;
using System.Collections.Generic;
using System.Text;

namespace CaptureTestConsole
{
    interface IFrameWriter
    {
        void WriteFrame(int width, int height, int rowPitch, IntPtr buffer);
    }
}
