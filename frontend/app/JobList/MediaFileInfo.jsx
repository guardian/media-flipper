import React from 'react';
import PropTypes from 'prop-types';
import HidableExpander from "../Common/HidableExpander.jsx";
import BytesFormatter from "../Common/BytesFormatter.jsx";
import css from './MediaFileInfo.css';

class MediaFileInfo extends React.Component {
    static propTypes = {
        fileInfo: PropTypes.object,  //see models/fileformat.go
        jobId: PropTypes.string.isRequired,
        initialExpanderState: PropTypes.bool
    };

    constructor(props) {
        super(props);

        this.state = {
            loading: false,
            lastError: null,
            fileInfo: null,
            explanatoryText: null
        }
    }

    getFormatAnalysis(){
        if(!this.state.fileInfo && !this.props.fileInfo) return null;
        if(this.state.fileInfo && this.state.fileInfo.hasOwnProperty("formatAnalysis")) return this.state.fileInfo.formatAnalysis;
        if(this.props.fileInfo && this.props.fileInfo.hasOwnProperty("formatAnalysis")) return this.props.fileInfo.formatAnalysis;
        return null;
    }

    componentDidMount() {
        if(!this.props.fileInfo) this.loadData();
    }

    async loadData() {
        const url = "/api/analysis/get?forId=" + this.props.jobId;
        const response = await fetch(url);
        if(response.status===200) {
            const content = await response.json();
            this.setState({loading: false, fileInfo: content.entry})
        } else if(response.status===404) {
            response.body.cancel();
            this.setState({loading: false, fileInfo: null, explanatoryText: "No media information available"})
        } else {
            response.body.cancel();
            this.setState({loading: false, lastError: response.statusText});
        }
    }

    render() {
        const filedata = this.getFormatAnalysis();
        if(!filedata){
            if(this.state.explanatoryText){
                return <HidableExpander headerText={this.state.explanatoryText}/>
            } else {
                return null;
            }
        }

        const showheader = filedata.format_long_name + ", " + Math.round(filedata.duration) + " seconds";
        return <HidableExpander headerText={showheader} initialExpanderState={this.props.initialExpanderState}>
            <table className="media-file-info">
                <tbody>
                <tr>
                    <td className="media-file-info right">Streams</td>
                    <td className="media-file-info left">{filedata.nb_streams}</td>
                </tr>
                <tr>
                    <td className="media-file-info right">Start time</td>
                    <td className="media-file-info left">{filedata.start_time}</td>
                </tr>
                <tr>
                    <td className="media-file-info right">Duration</td>
                    <td className="media-file-info left">{filedata.duration}s</td>
                </tr>
                <tr>
                    <td className="media-file-info right">Bitrate</td>
                    <td className="media-file-info left">{filedata.bit_rate}</td>
                </tr>
                <tr>
                    <td className="media-file-info right">File size</td>
                    <td className="media-file-info left"><BytesFormatter value={filedata.size}/></td>
                </tr>
                <tr>
                    <td className="media-file-info right">Probe score</td>
                    <td className="media-file-info left">{filedata.probe_score}</td>
                </tr>
                </tbody>
            </table>
        </HidableExpander>
    }
}

export default MediaFileInfo;