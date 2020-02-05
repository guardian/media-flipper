import React from 'react';
import PropTypes from 'prop-types';
import ThumbnailPreview from "./ThumbnailPreview.jsx";

class MediaPreview extends React.Component {
    static propTypes = {
        fileId: PropTypes.string,
        onClick: PropTypes.func,
        clickable: PropTypes.bool,
        className: PropTypes.string.isRequired
    };

    constructor(props) {
        super(props);

        this.state = {
            fileMeta: {},
            lastError: null
        }
    }

    setStatePromise(newState) {
        return new Promise((resolve, reject)=>this.setState(newState, ()=>resolve()))
    }

    async updateMeta() {
        if(this.props.fileId!==null && this.props.fileId!=="00000000-0000-0000-0000-000000000000") {
            const result = await fetch("/api/file/get?forId=" + this.props.fileId);

            if (result.status === 200) {
                const content = await result.json();
                return this.setStatePromise({fileMeta: content.entry});
            } else {
                const content = await result.text();
                return this.setStatePromise({lastError: content});
            }
        } else {
            return new Promise((resolve,reject)=>reject("nothing to get"));
        }
    }

    componentDidMount() {
        this.updateMeta().then(()=>{
            console.log("updated file metadata: ")
        })
    }

    componentDidUpdate(prevProps, prevState, snapshot) {
        if(prevProps.fileId!==this.props.fileId) {
            this.updateMeta();
        }
    }

    render() {
        if(this.props.fileId && this.props.fileId!=="00000000-0000-0000-0000-000000000000"){
            const contentStreamUrl = "/api/file/content?forId=" + this.props.fileId;
            if(this.state.fileMeta.hasOwnProperty("mimeType")){
                if(this.state.fileMeta.mimeType.startsWith("video/")){
                    return <span>
                        <video src={contentStreamUrl} controls={true} className={this.props.className}/><br/>
                        <a href={contentStreamUrl} download>Download content...</a>
                    </span>
                } else if(this.state.fileMeta.mimeType.startsWith("audio/")){
                    return <audio src={contentStreamUrl} controls={true} className={this.props.className}/>
                } else if(this.state.fileMeta.mimeType.startsWith("image/")){
                    return <ThumbnailPreview className={this.props.className} clickable={this.props.clickable} onClick={this.props.onClick} fileId={this.props.fileId}/>
                } else {
                    return <p className="information">Don't know how to display content of type {this.state.fileMeta.mimeType}</p>
                }
            } else {
                return <p className="error-text">File had no type information</p>
            }
            // return <table><tbody>
            // {Object.keys(this.state.fileMeta).map(k=><tr key={k}><td>{k}</td><td>{this.state.fileMeta[k]}</td></tr>)}
            // </tbody>
            // </table>
        } else {
            return null;
        }
    }
}

export default MediaPreview;