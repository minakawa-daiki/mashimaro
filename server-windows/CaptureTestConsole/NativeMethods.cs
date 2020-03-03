using System;
using System.Runtime.InteropServices;

namespace CaptureTestConsole
{
    internal enum VideoFormat
    {
        YCbCr411_8bit,      // Pixelgroup: 4 pixels in 6  bytes
        RGB_8bit,           // Pixelgroup: 1 pixel  in 3  bytes
        RGBA_8bit,          // Pixelgroup: 1 pixel  in 4  bytes
        BGR_8bit,           // Pixelgroup: 1 pixel  in 3  bytes
        BGRA_8bit,          // Pixelgroup: 1 pixel  in 4  bytes
        RGB_10bit,          // Pixelgroup: 4 pixels in 15 bytes
        RGBA_10bit,         // Pixelgroup: 1 pixel  in 5  bytes
        BGR_10bit,          // Pixelgroup: 4 pixels in 15 bytes
        BGRA_10bit,         // Pixelgroup: 1 pixel  in 5  bytes
        RGB_12bit,          // Pixelgroup: 2 pixels in 9  bytes
        RGBA_12bit,         // Pixelgroup: 1 pixel  in 6  bytes
        BGR_12bit,          // Pixelgroup: 2 pixels in 9  bytes
        BGRA_12bit,         // Pixelgroup: 1 pixel  in 6  bytes
        RGB_16bit,          // Pixelgroup: 1 pixel  in 6  bytes
        RGBA_16bit,         // Pixelgroup: 1 pixel  in 8  bytes
        BGR_16bit,          // Pixelgroup: 1 pixel  in 6  bytes
        BGRA_16bit,         // Pixelgroup: 1 pixel  in 8  bytes
        RGBp_5bit,          // Pixelgroup: 1 pixel  in 3  bytes (RFC4421)
        RGpB_5bit,          // Pixelgroup: 1 pixel  in 3  bytes (RFC4421)
        RpGB_5bit,          // Pixelgroup: 1 pixel  in 3  bytes (RFC4421)
        BGRp_5bit,          // Pixelgroup: 1 pixel  in 3  bytes (RFC4421)
        BGpR_5bit,          // Pixelgroup: 1 pixel  in 3  bytes (RFC4421)
        BpGR_5bit,          // Pixelgroup: 1 pixel  in 3  bytes (RFC4421)
    }
    
    internal static class NativeMethods
    {
        [DllImport("rtpvideo.dll")]
        internal static extern IntPtr RtpVideoTx_new(int sock, VideoFormat format);
        
        [DllImport("rtpvideo.dll")]
        internal static extern int RtpVideoTx_addDestination(IntPtr v, [MarshalAs(UnmanagedType.LPStr), In] string host, uint port);
        
        [DllImport("rtpvideo.dll")]
        internal static extern int RtpVideoTx_setSSRC(IntPtr v, uint ssrc);
        
        [DllImport("rtpvideo.dll")]
        internal static extern int RtpVideoTx_beginFrame(IntPtr v, uint timestamp);
        
        [DllImport("rtpvideo.dll")]
        internal static extern int RtpVideoTx_addLine(IntPtr v, uint lineNo, uint pixelOffset, uint length, IntPtr buffer, ulong flags);
    }
}